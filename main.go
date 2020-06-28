package main

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/koron-go/srcdom"
)

func main() {
	err := run(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func goEnvRoot(ctx context.Context) (string, error) {
	c := exec.CommandContext(ctx, "go", "env", "GOROOT")
	b, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

type walker struct {
	ctx    context.Context
	srcdir string

	procPkg func(string, string, *srcdom.Package) error
}

func (w *walker) walk(path string, info os.FileInfo, err error) error {
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if info != nil && info.IsDir() {
		return w.walkDir(path, info, err)
	}
	return nil
}

func (w *walker) ignoreTests(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}

func (w *walker) walkDir(path string, info os.FileInfo, err error) error {
	switch info.Name() {
	case "internal", "testdata", "vendor":
		return filepath.SkipDir
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, w.ignoreTests, 0)
	if err != nil {
		return fmt.Errorf("parser failure: %w", err)
	}
	p := &srcdom.Parser{}
	pkgname := ""
	for n, pkg := range pkgs {
		if n == "main" || strings.HasSuffix(n, "_test") {
			continue
		}
		if pkgname != "" {
			log.Printf("conflict package names: first=%s other=%s", pkgname, n)
		}
		pkgname = n
		for _, f := range pkg.Files {
			err := p.ScanFile(f)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
		}
	}
	return w.walkPkg(path, pkgname, p.Package)
}

func (w *walker) walkPkg(path, name string, pkg *srcdom.Package) error {
	if pkg == nil || w.procPkg == nil {
		return nil
	}
	return w.procPkg(path, name, pkg)
}

func (w *walker) listPublic(path, name string, pkg *srcdom.Package) error {
	rel, err := filepath.Rel(w.srcdir, path)
	if err != nil {
		log.Printf("filepath.Rel failed but ignored: %s", err)
		rel = path
	}
	fmt.Printf("package: %s\n", filepath.ToSlash(rel))
	for _, typ := range pkg.Types {
		if !typ.IsPublic() {
			continue
		}
		fmt.Printf("type: %s.%s\n", name, typ.Name)
	}
	for _, fn := range pkg.Funcs {
		if !fn.IsPublic() {
			continue
		}
		fmt.Printf("func: %s.%s\n", name, fn.Name)
	}
	for _, v := range pkg.Values {
		if !v.IsPublic() {
			continue
		}
		fmt.Printf("value: %s.%s\n", name, v.Name)
	}
	return nil
}

var ignorePackages = []string{
	// keep in sort
	"ast",
	"build",
	"builtin",
	"constant",
	"doc",
	"driver",
	"dwarf",
	"elf",
	"heap",
	"importer",
	"macho",
	"parse",
	"parser",
	"pe",
	"plan9obj",
	"printer",
	"runtime",
	"scanner",
	"syscall",
	"token",
	"types",
	"user",
}

func shouldSkip(name string) bool {
	x := sort.SearchStrings(ignorePackages, name)
	if x < len(ignorePackages) {
		return ignorePackages[x] == name
	}
	return false
}

func (w *walker) genVimSyntax(path, name string, pkg *srcdom.Package) error {
	n := len(pkg.Types)
	if n == 0 || shouldSkip(name) {
		fmt.Printf("\" skipped %s package: mark as IGNORE\n", name)
		return nil
	}

	list := make([]string, 0, n)
	for _, typ := range pkg.Types {
		if typ.IsPublic() {
			list = append(list, typ.Name)
		}
	}
	if len(list) == 0 {
		fmt.Printf("\" skipped %s package: no public symbols\n", name)
		return nil
	}
	sort.Strings(list)

	b := &strings.Builder{}
	fmt.Fprintf(b, `syn match goExtraType /\<%s\.\(`, name)
	for i, typname := range list {
		if i > 0 {
			b.WriteString(`\|`)
		}
		b.WriteString(typname)
	}
	b.WriteString(`\)\>/`)
	fmt.Println(b.String())
	return nil
}

func (w *walker) countPublic(p *srcdom.Package) int {
	if p == nil {
		return 0
	}
	cnt := 0
	for _, typ := range p.Types {
		if typ.IsPublic() {
			cnt++
		}
	}
	for _, fn := range p.Funcs {
		if fn.IsPublic() {
			cnt++
		}
	}
	for _, v := range p.Values {
		if v.IsPublic() {
			cnt++
		}
	}
	return cnt
}

func run(ctx context.Context) error {
	var (
		mode string
		root string
	)
	flag.StringVar(&mode, "mode", "list", `how process packages (list, syntax)`)
	flag.StringVar(&root, "root", "", `root dir to scan (default $GOROOT)`)
	flag.Parse()
	if root == "" {
		goroot, err := goEnvRoot(ctx)
		if err != nil {
			return err
		}
		root = goroot
	}
	srcdir := filepath.Join(root, "src")
	w := walker{
		ctx:    ctx,
		srcdir: srcdir,
	}
	switch mode {
	case "syntax":
		w.procPkg = w.genVimSyntax
	default:
		w.procPkg = w.listPublic
	}
	return filepath.Walk(srcdir, w.walk)
}
