package agent

import (
	"context"
	"testing"
	"time"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/mask"
	"github.com/oz-fatma/kontrata/backend/internal/pdf"
	"os"
	"path/filepath"
)

func TestChunkA_LiveArgosMEGEP(t *testing.T) {
	if testing.Short() {
		t.Skip("kısa test modunda canlı LLM atlanır")
	}
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)
	client := liveLLMClient(t)

	path := filepath.Clean(filepath.Join("..", "..", "..", "testdata", "sozlesmeler", "argos-megep.pdf"))
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	defer func() { _ = f.Close() }()
	pages, err := pdf.ExtractText(f)
	if err != nil {
		t.Fatalf("metin: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ctx = llm.WithMeta(ctx, llm.Meta{Agent: llm.AgentReaderChunkA})

	r := &Reader{LLM: client, ContractID: "argos-chunk-a", DumpDir: t.TempDir()}
	spec := chunkSpecs[0]
	masked := mask.Apply(joinPages(pages))
	out := r.fetchChunk(ctx, spec, masked.Text)
	if out.genErr != nil {
		t.Fatalf("model: %v", out.genErr)
	}
	if out.data == nil {
		t.Fatal("parça A json cozulemedi")
	}

	assertArgosChunkAShape(t, out.data)
	assertArgosOdaKontenjanlari(t, out.data)
}
