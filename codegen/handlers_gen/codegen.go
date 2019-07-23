package main

//go build codegen.go && codegen.exe ../api.go ../api_handlers.go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

// код писать тут

func stringInSlice(a string, list []string) (bool, int) {
	for i, b := range list {
		if b == a {
			return true, i
		}
	}
	return false, -1
}

type tpl struct {
	MethodName     string
	ApiUserCreate  string
	ApiUserProfile string
	StructVal      string
	Structtype     string
	Url            string
	Auth           bool
	Method         string
}

type tplParamServe struct {
	MethodName string
	Structtype string
	Url        string
}
type tplParamRequired struct {
	Field string
}
type tplParamEnum struct {
	Values string
	Field  string
}
type tplParamWrapper struct {
	MethodName string
	This       string
	Structtype string
	Url        string
	Auth       bool
	Method     string
}
type tplParamServe1 struct {
	StructType string
}
type tplParamMin struct {
	MinVal string
	Field  string
}
type tplParamMax struct {
	MaxVal string
	Field  string
}
type tplParamDefault struct {
	DefaultVal string
	Field      string
}

var (
	serveHTTP1 = template.Must(template.New("serveHTTP").Parse(
		`func  (this *{{.StructType}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path{
	`))
	serveHTTPurl = template.Must(template.New("serveHTTP").Parse(`
		case "{{.Url}}":
			this.Wrapper{{.MethodName}}(w,r)
	`))
	serveHTTP3 = `
		default:
			a, _ := json.Marshal(CR{"error": "unknown method"})
			w.WriteHeader(http.StatusNotFound)
			w.Write(a)
			return
		}
	}
	`  //))
	required = template.Must(template.New("required").Parse(`
		if {{.Field}} == "" {
			a, _ := json.Marshal(CR{
				"error": strings.ToLower("{{.Field}}")+ " must me not empty",
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(a)
			return
		}
	`))
	tplEnum = template.Must(template.New("enum").Parse(
		`
		if ok, _ := stringInSlice({{.Field}}, strings.Split("{{.Values}}", ", ") ); !ok {
			a, _ := json.Marshal(CR{
				"error": strings.ToLower("{{.Field}}")+" must be one of [{{.Values}}]",
			})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(a)
			return
	}
	`))

	tplAuth = `	if r.Header.Get("X-Auth") != "100500" {
			a, _ := json.Marshal(CR{"error": "unauthorized"})
			w.WriteHeader(http.StatusForbidden)
			w.Write(a)
			return
	}
	`

	tplMinInt = template.Must(template.New("minInt").Parse(
		`
		if {{.Field}} < {{.MinVal}} {
			a, _ := json.Marshal(CR{"error": strings.ToLower("{{.Field}}")+" must be >= {{.MinVal}}"})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(a)
			return
		}
		
		`))
	tplMinStr = template.Must(template.New("minStr").Parse(
		`
			if len({{.Field}}) < {{.MinVal}} {
				a, _ := json.Marshal(CR{"error": strings.ToLower("{{.Field}}")+" len must be >= {{.MinVal}}"})
				w.WriteHeader(http.StatusBadRequest)
				w.Write(a)
				return
			}
			
			`))
	tplMaxStr = template.Must(template.New("maxStr").Parse(
		`
		if len({{.Field}}) > {{.MaxVal}} {
			a, _ := json.Marshal(CR{"error": strings.ToLower("{{.Field}}")+" len must be <= {{.MaxVal}}"})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(a)
			return
		}	
		`))
	tplMaxInt = template.Must(template.New("maxInt").Parse(
		`
		if {{.Field}} > {{.MaxVal}} {
			a, _ := json.Marshal(CR{"error": strings.ToLower("{{.Field}}")+" must be <= {{.MaxVal}}"})
			w.WriteHeader(http.StatusBadRequest)
			w.Write(a)
			return
		}	
		`))
	tplDefault = template.Must(template.New("default").Parse(
		`
			if {{.Field}} == "" {
				{{.Field}} = "{{.DefaultVal}}"
			}	
			`))
)

type params struct {
	MethodName string
	//StructType string
	Url string
}

type ApiParamsToCheck struct {
	Code          string
	ParamsToCheck []string
	Type          string
	Name2         string
}

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out,
		`import (
		"encoding/json"
		"net/http"
		"strconv"
		"strings"
	)
	type Result struct{
		error string
		response CR
	}
	func stringInSlice(a string, list []string) (bool, int) {
		for i, b := range list {
			if b == a {
				return true, i
			}
		}
		return false, -1
	}
	`)

	structFields := map[string]map[string]ApiParamsToCheck{}
	var structs map[string][]params
	structs = map[string][]params{}

CON:
	for _, f := range node.Decls {
		g, ok := f.(*ast.FuncDecl)
		// Struct decl
		if !ok {
			g, ok := f.(*ast.GenDecl)

			if !ok {
				fmt.Printf("SKIP %T is not GenDecl and Func \n", f)
				continue
			}
			for _, spec := range g.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}

				fmt.Printf("Is Struct\n")

				validParam := map[string]ApiParamsToCheck{}
				for _, field := range currStruct.Fields.List {
					if field.Tag == nil {
						fmt.Println("Has no tags")
						continue
					}
					tag := field.Tag.Value
					fulltag := strings.Split(tag, ":")
					tag = strings.Replace(strings.Replace(fulltag[1], "`", "", -1), "\"", "", -1)
					buf := bytes.NewBufferString("")

					if fulltag[0] == "`apivalidator" {
						apiparams := strings.Split(tag, ",")
						for _, param := range apiparams {

							flagDefault := false
							name := ""
							switch {
							case strings.HasPrefix(param, "required"):
								required.Execute(buf, tplParamRequired{field.Names[0].Name})

							case strings.HasPrefix(param, "enum"):
								e := strings.TrimPrefix(param, "enum=")
								enumVal := strings.Replace(e, "|", ", ", -1)
								tplEnum.Execute(buf, tplParamEnum{enumVal, field.Names[0].Name})

							case strings.HasPrefix(param, "min"):
								minVal := strings.TrimPrefix(param, "min=")

								if field.Type.(*ast.Ident).Name == "int" {
									tplMinInt.Execute(buf, tplParamMin{minVal, field.Names[0].Name})
								}
								if field.Type.(*ast.Ident).Name == "string" {
									tplMinStr.Execute(buf, tplParamMin{minVal, field.Names[0].Name})
								}
							case strings.HasPrefix(param, "max"):
								maxVal := strings.TrimPrefix(param, "max=")

								if field.Type.(*ast.Ident).Name == "int" {
									tplMaxInt.Execute(buf, tplParamMax{maxVal, field.Names[0].Name})
								}
								if field.Type.(*ast.Ident).Name == "string" {
									tplMaxStr.Execute(buf, tplParamMax{maxVal, field.Names[0].Name})
								}
							case strings.HasPrefix(param, "paramname"):
								name = strings.TrimPrefix(param, "paramname=")
							case strings.HasPrefix(param, "default"):
								defaultVal := strings.TrimPrefix(param, "default=")
								buf2 := bytes.NewBufferString("")
								tplDefault.Execute(buf2, tplParamDefault{defaultVal, field.Names[0].Name})
								code := buf2.String() + buf.String()

								validParam[field.Names[0].Name] = ApiParamsToCheck{
									Code:  code,
									Type:  field.Type.(*ast.Ident).Name,
									Name2: name,
								}
								flagDefault = true
							}

							if !flagDefault {
								validParam[field.Names[0].Name] = ApiParamsToCheck{
									Code:  buf.String(),
									Type:  field.Type.(*ast.Ident).Name,
									Name2: name,
									//g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
								}
							}

							structFields[currType.Name.Name] = validParam
							fmt.Println(structFields)
						}
					}

				}
			}
			continue CON
		}

		//Func decl
		if g.Doc == nil {
			fmt.Printf("SKIP func %#v doesnt have comments\n", g.Name.Name)
			continue
		}

		needCodegen := false
		var com string
		for _, comment := range g.Doc.List {
			needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
			com = strings.TrimPrefix(comment.Text, "// apigen:api")
		}
		if !needCodegen {
			fmt.Printf("SKIP func %#v doesnt have apigen mark\n", g.Name.Name)
			continue
		}

		//Создание wrapper
		fmt.Println("Creating wrapper")
		var args map[string]interface{}
		json.Unmarshal([]byte(com), &args)

		Auth := false
		if args["auth"] != nil {
			Auth = args["auth"].(bool)
		}
		Method := ""
		if args["method"] != nil {
			Method = args["method"].(string)
		}
		URL := ""
		if args["url"] != nil {
			URL = args["url"].(string)
		}

		strType := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		method := params{
			MethodName: g.Name.Name,
			Url:        URL,
		}
		structs[strType] = append(structs[strType], method)

		fmt.Println("Checking params of " + g.Name.Name)
		fmt.Fprintln(out,
			`func (`+
				g.Recv.List[0].Names[0].Name+
				`*`+g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name+
				") Wrapper"+g.Name.Name+
				`( w http.ResponseWriter, r *http.Request) {
			`)
		// проверка метода
		if Method == "POST" {
			fmt.Fprintln(out, `
				if r.Method != "POST"{
					a, _ := json.Marshal(CR{"error": "bad method",})
					w.WriteHeader(http.StatusNotAcceptable)
					w.Write(a)
					return
				} `)
		}

		//проверка авторизации
		if Auth {
			fmt.Fprintln(out, tplAuth)
		}

		//валидация параметров
		structType := g.Type.Params.List[1].Type.(*ast.Ident).Name
		fmt.Fprintln(out, `var err error`)
		for field, tocheck := range structFields[structType] {
			fieldURL := field
			if tocheck.Name2 != "" {
				fieldURL = tocheck.Name2
			}

			switch tocheck.Type {
			case "string":
				fmt.Fprintln(out, `var `+field+` string`)
				if Method == "POST" {
					fmt.Fprintln(out,
						field+` = r.FormValue(strings.ToLower("`+fieldURL+`"))`)
				}
				if Method == "GET" {
					fmt.Fprintln(out, field+` = r.URL.Query().Get(strings.ToLower("`+fieldURL+`)) `)
				}
				if Method == "" {
					fmt.Fprintln(out, `	switch r.Method {
				case "GET":
					`+field+` = r.URL.Query().Get(strings.ToLower("`+fieldURL+`"))
				case "POST":
					`+field+` = r.FormValue(strings.ToLower("`+fieldURL+`"))
				}`)
				}

			case "int":
				fmt.Fprintln(out, `var `+field+` int`)

				if Method == "POST" {
					fmt.Fprintln(out, field+`, err = strconv.Atoi(r.FormValue(strings.ToLower("`+fieldURL+`")))`)
				}
				if Method == "GET" {
					fmt.Fprintln(out, field+", err = strconv.Atoi(r.URL.Query().Get(strings.ToLower(\""+fieldURL+"\"))) ")
				}
				if Method == "" {
					fmt.Fprintln(out, `	switch r.Method {
				case "GET":
					`+field+`, err = strconv.Atoi(r.URL.Query().Get("`+fieldURL+`"))
				case "POST":
					`+field+`, err = strconv.Atoi(r.FormValue(strings.ToLower("`+fieldURL+`")))
				}`)
				}
				fmt.Fprintln(out,
					`			
					if err!= nil {
							a, _ := json.Marshal(CR{
						"error": strings.ToLower("`+fieldURL+`") + " must be int",
						})
						w.WriteHeader(http.StatusBadRequest)
						w.Write(a)
						return
					}`)
			}
			fmt.Fprintln(out, structFields[structType][field].Code)
		}

		//заполнение структуры
		fmt.Fprintln(out, `
		var in `+structType+`
		in = `+structType+`{`)

		for field, _ := range structFields[structType] {
			fmt.Fprintln(out, field+`: `+field+`,`)
		}
		fmt.Fprintln(out, `}
		res, err := srv.`+g.Name.Name+`(r.Context(), in)
		if err!= nil{
			_, ok := err.(ApiError)
			if ok {
				a, _ := json.Marshal(CR{"error": err.(ApiError).Err.Error()})
				w.WriteHeader(err.(ApiError).HTTPStatus)
				w.Write(a)
			} else {
				a, _ := json.Marshal(CR{"error": "bad user"})
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(a)
			}
			return
		}
		a, _ := json.Marshal(CR{"error": "", "response": res})
		w.WriteHeader(http.StatusOK)
		w.Write(a)
		return
		`)

		fmt.Fprintln(out, "}")
	}

	// Serve HTTP
	fmt.Println("Creating serveHTTP for ")
	fmt.Println(structs)
	for structType, s := range structs {
		// fmt.Println(structType)
		// fmt.Println(s)
		serveHTTP1.Execute(out, tplParamServe1{structType})
		for _, method := range s {
			serveHTTPurl.Execute(out, tplParamServe{method.MethodName, structType, method.Url})
		}
		fmt.Fprintln(out, serveHTTP3)

	}
}
