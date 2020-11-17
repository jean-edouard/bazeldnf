package bazel

import (
	"fmt"
	"io/ioutil"
	"path"
	"sort"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
	"github.com/rmohr/bazeldnf/pkg/api"
)

type Artifact struct {
	rule *build.Rule
}

func LoadWorkspace(path string) (*build.File, error) {
	workspaceData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE orig: %v", err)
	}
	workspace, err := build.ParseWorkspace(path, workspaceData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE orig: %v", err)
	}
	return workspace, nil
}

func LoadBuild(path string) (*build.File, error) {
	buildfileData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE orig: %v", err)
	}
	buildfile, err := build.ParseBuild(path, buildfileData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE orig: %v", err)
	}
	return buildfile, nil
}

func WriteBuild(dryRun bool, buildfile *build.File, path string) error {
	if dryRun {
		fmt.Println(build.FormatString(buildfile))
		return nil
	}
	return ioutil.WriteFile(path, build.Format(buildfile), 0666)
}

func WriteWorkspace(dryRun bool, workspace *build.File, path string) error {
	if dryRun {
		fmt.Println(build.FormatString(workspace))
		return nil
	}
	return ioutil.WriteFile(path, build.Format(workspace), 0666)
}

func AddRPMS(workspace *build.File, pkgs []*api.Package) {

	rpms := map[string]*rpmRule{}

	for _, rule := range workspace.Rules("rpm") {
		rpms[rule.Name()] = &rpmRule{rule}
	}

	for _, pkg := range pkgs {
		rule := rpms[pkg.String()]
		if rule == nil {
			call := &build.CallExpr{X: &build.Ident{Name: "rpm"}}
			rule = &rpmRule{&build.Rule{call, ""}}
			rpms[pkg.String()] = rule
		}
		rule.SetName(pkg.String())
		rule.SetSHA256(pkg.Checksum.Text)
		urls := rule.URLs()
		if len(urls) == 0 {
			rule.SetURLs(pkg.Repository.Mirrors, pkg.Location.Href)
		}
	}

	rules := []*rpmRule{}
	for _, rule := range rpms {
		rules = append(rules, rule)
	}

	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Name() < rules[j].Name()
	})

	workspace.DelRules("rpm", "")
	for _, rule := range rules {
		workspace.Stmt = edit.InsertAtEnd(workspace.Stmt, rule.Call)
	}
}

func AddTree(name string, buildfile *build.File, pkgs []*api.Package, files []string) {
	rpmtrees := map[string]*rpmTree{}

	for _, rule := range buildfile.Rules("rpmtree") {
		rpmtrees[rule.Name()] = &rpmTree{rule}
	}

	rpms := []string{}
	for _, pkg := range pkgs {
		rpms = append(rpms, "@"+pkg.Name+"//rpm")
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i] < files[j]
	})

	rule := rpmtrees[name]
	if rule == nil {
		call := &build.CallExpr{X: &build.Ident{Name: "rpmtree"}}
		rule = &rpmTree{&build.Rule{call, ""}}
		rpmtrees[name] = rule
	}
	rule.SetName(name)
	rule.SetRPMs(rpms)
	rule.SetFiles(files)

	rules := []*rpmTree{}
	for _, rule := range rpmtrees {
		rules = append(rules, rule)
	}

	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Name() < rules[j].Name()
	})


	for _, rule := range rules {
		buildfile.Stmt = edit.InsertAtEnd(buildfile.Stmt, rule.Call)
	}
}

type rpmRule struct {
	*build.Rule
}

func (r *rpmRule) URLs() []string {
	if urlsAttr := r.Rule.Attr("urls"); urlsAttr != nil {
		if len(urlsAttr.(*build.ListExpr).List) > 0 {
			urls := []string{}
			for _, expr := range urlsAttr.(*build.ListExpr).List {
				urls = append(urls, expr.(*build.StringExpr).Value)
			}
			return urls
		}
	}
	return nil
}

func (r *rpmRule) SetURLs(urls []string, href string) {
	urlsAttr := []build.Expr{}
	for _, url := range urls {
		urlsAttr = append(urlsAttr, &build.StringExpr{Value: path.Join(url, href)})
	}
	r.Rule.SetAttr("urls", &build.ListExpr{List: urlsAttr})
}

func (r *rpmRule) SetName(name string) {
	r.Rule.SetAttr("name", &build.StringExpr{Value: name})
}

func (r *rpmRule) SetSHA256(sha256 string) {
	r.Rule.SetAttr("sha256", &build.StringExpr{Value: sha256})
}

type rpmTree struct {
	*build.Rule
}

func (r *rpmTree) SetName(name string) {
	r.Rule.SetAttr("name", &build.StringExpr{Value: name})
}

func (r *rpmTree) RPMs() []string {
	if rpmAttrs := r.Rule.Attr("rpms"); rpmAttrs != nil {
		if len(rpmAttrs.(*build.ListExpr).List) > 0 {
			rpms := []string{}
			for _, expr := range rpmAttrs.(*build.ListExpr).List {
				rpms = append(rpms, expr.(*build.StringExpr).Value)
			}
			return rpms
		}
	}
	return nil
}

func (r *rpmTree) SetRPMs(rpms []string) {
	rpmsAttr := []build.Expr{}
	for _, rpm := range rpms {
		rpmsAttr = append(rpmsAttr, &build.StringExpr{Value: rpm})
	}
	r.Rule.SetAttr("rpms", &build.ListExpr{List: rpmsAttr})
}

func (r *rpmTree) SetFiles(files []string) {
	filesAttr := []build.Expr{}
	for _, file := range files {
		filesAttr = append(filesAttr, &build.StringExpr{Value: file})
	}
	r.Rule.SetAttr("files", &build.ListExpr{List: filesAttr})
}