// Gera assets/top-langs.svg com as linguagens mais usadas, incluindo repositórios privados.
//
// Usa o GitHub CLI já autenticado (gh), então nenhum token é salvo no repositório.
// Uso (na raiz do repositório do perfil): go run ./tools/toplangs
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const owner = "BrunoBarrosR"

// Repositórios que distorcem a estatística (venv commitado, código gerado etc.).
var excludeRepos = map[string]bool{
	"projeto_cadastro_django": true,
}

// Itens que não são linguagens de programação para este card.
var hideLangs = map[string]bool{
	"HTML": true,
	"CSS":  true,
	"Bru":  true,
}

var colors = map[string]string{
	"Go":         "#00ADD8",
	"Java":       "#b07219",
	"Python":     "#3572A5",
	"JavaScript": "#f1e05a",
	"TypeScript": "#3178c6",
	"Shell":      "#89e051",
	"PowerShell": "#012456",
}

const maxLangs = 8

type repo struct {
	Name   string `json:"name"`
	IsFork bool   `json:"isFork"`
}

func gh(args ...string) ([]byte, error) {
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run() error {
	out, err := gh("repo", "list", owner, "--limit", "200", "--source", "--no-archived", "--json", "name,isFork")
	if err != nil {
		return err
	}
	var repos []repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return err
	}

	totals := map[string]int64{}
	for _, r := range repos {
		if r.IsFork || excludeRepos[r.Name] {
			continue
		}
		out, err := gh("api", fmt.Sprintf("repos/%s/%s/languages", owner, r.Name))
		if err != nil {
			return err
		}
		var langs map[string]int64
		if err := json.Unmarshal(out, &langs); err != nil {
			return err
		}
		for l, b := range langs {
			if !hideLangs[l] {
				totals[l] += b
			}
		}
	}

	type item struct {
		name  string
		bytes int64
	}
	var items []item
	var sum int64
	for l, b := range totals {
		items = append(items, item{l, b})
		sum += b
	}
	if sum == 0 {
		return fmt.Errorf("nenhuma linguagem encontrada")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].bytes > items[j].bytes })
	if len(items) > maxLangs {
		items = items[:maxLangs]
		sum = 0
		for _, it := range items {
			sum += it.bytes
		}
	}

	var bar, legend strings.Builder
	const barX, barW = 25.0, 350.0
	x := barX
	for i, it := range items {
		c := colors[it.name]
		if c == "" {
			c = "#8b949e"
		}
		pct := float64(it.bytes) * 100 / float64(sum)
		w := barW * float64(it.bytes) / float64(sum)
		fmt.Fprintf(&bar, `<rect x="%.2f" y="0" width="%.2f" height="8" fill="%s"/>`, x, w, c)
		x += w

		col, row := i%2, i/2
		lx := 25 + col*180
		ly := 14 + row*25
		fmt.Fprintf(&legend, `<g transform="translate(%d,%d)"><circle cx="5" cy="6" r="5" fill="%s"/><text x="15" y="10" class="t">%s %.2f%%</text></g>`,
			lx, ly, c, it.name, pct)
	}

	rows := (len(items) + 1) / 2
	height := 85 + rows*25
	svg := fmt.Sprintf(`<svg width="400" height="%d" viewBox="0 0 400 %d" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Most Used Languages">
<style>.h{font:600 18px 'Segoe UI',Ubuntu,Sans-Serif;fill:#c792ea}.t{font:400 12px 'Segoe UI',Ubuntu,Sans-Serif;fill:#7fdbca}</style>
<rect width="400" height="%d" rx="4.5" fill="#011627"/>
<text x="25" y="35" class="h">Most Used Languages</text>
<g transform="translate(0,55)"><clipPath id="c"><rect x="25" y="0" width="350" height="8" rx="4"/></clipPath><g clip-path="url(#c)">%s</g></g>
<g transform="translate(0,75)">%s</g>
</svg>
`, height, height, height, bar.String(), legend.String())

	dst := filepath.Join("assets", "top-langs.svg")
	if err := os.MkdirAll("assets", 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, []byte(svg), 0o644); err != nil {
		return err
	}
	for _, it := range items {
		fmt.Printf("%-12s %6.2f%%\n", it.name, float64(it.bytes)*100/float64(sum))
	}
	return nil
}
