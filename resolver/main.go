// Entrypoint local que simula o Lambda de resolução: lê screen definition + payload +
// catálogo, chama ResolveScreen e persiste o JSON resolvido via ScreenStore.
// Em produção este main vira o handler aws-lambda-go (lambda.Start) e o ScreenStore
// vira um S3 PutObject — a lógica de resolver.go não muda.
//
// Uso:
//   go run . -screen ../screens/example-cartao.json -payload payload.json \
//            -catalog ../catalog/schema.json -out ./out
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// ScreenStore abstrai o destino do JSON resolvido (S3 em produção, disco na POC).
type ScreenStore interface {
	Put(key string, data []byte) error
}

type LocalDirStore struct{ Dir string }

func (s LocalDirStore) Put(key string, data []byte) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, key), data, 0o644)
}

func run(screenPath, payloadPath, catalogPath string, store ScreenStore) error {
	catalogRaw, err := os.ReadFile(catalogPath)
	if err != nil {
		return err
	}
	catalog, err := ParseCatalog(catalogRaw)
	if err != nil {
		return err
	}

	screenRaw, err := os.ReadFile(screenPath)
	if err != nil {
		return err
	}
	def, err := ParseScreenDefinition(screenRaw)
	if err != nil {
		return err
	}

	payloadRaw, err := os.ReadFile(payloadPath)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}

	resolved, errs := ResolveScreen(def, payload, catalog)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "ERRO", e.Error())
		}
		return fmt.Errorf("resolução falhou com %d erro(s)", len(errs))
	}

	out, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s@%s.resolved.json", resolved.ScreenID, resolved.ScreenVersion)
	if err := store.Put(key, out); err != nil {
		return err
	}
	fmt.Println("resolvido:", key)
	return nil
}

func main() {
	screen := flag.String("screen", "", "caminho da screen definition")
	payload := flag.String("payload", "", "caminho do payload de variáveis (JSON objeto)")
	catalog := flag.String("catalog", "../catalog/schema.json", "caminho do catálogo")
	out := flag.String("out", "./out", "diretório de saída (simula o bucket)")
	flag.Parse()

	if *screen == "" || *payload == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*screen, *payload, *catalog, LocalDirStore{Dir: *out}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
