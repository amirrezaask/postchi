package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type envSource string

const (
	envSource_Plain   = "plain"
	envSource_Args    = "args"
	envSource_Process = "process"
)

type Var struct {
	Source envSource `json:"source" yaml:"source"`
	Name   string    `json:"name" yaml:"name"`
	Index  int       `json:"index" yaml:"index"`
	Value  string    `json:"value" yaml:"value"`
}
type request struct {
	Method  string            `yaml:"method" json:"method"`
	Route   string            `json:"route" yaml:"route"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Body    string            `json:"body" yaml:"body"`
	Query   map[string]string `json:"query" yaml:"query"`
}

type Context struct {
	ProcessedVars map[string]string
	RawVars       map[string]Var     `json:"vars" yaml:"vars"`
	Defaults      request            `json:"defaults" yaml:"defaults"`
	Requests      map[string]request `json:"requests" yaml:"requests"`
}

func newContext(args []string, decoderFunc func(v any) error) (Context, error) {
	c := Context{ProcessedVars: map[string]string{}}
	err := decoderFunc(&c)
	if err != nil {
		return Context{}, fmt.Errorf("cannot decode using decoderFunc: %w", err)
	}
	for k, v := range c.RawVars {
		switch v.Source {
		case envSource_Process:
			envValue := os.Getenv(v.Name)
			if envValue == "" {
				envValue = v.Value
			}
			c.ProcessedVars[k] = envValue
		case envSource_Args:
			if len(args) > v.Index && args[v.Index] != "" {
				c.ProcessedVars[k] = args[v.Index]
			} else {
				c.ProcessedVars[k] = v.Value
			}
		case envSource_Plain:
			c.ProcessedVars[k] = v.Value
		}
	}

	return c, nil
}

const DEFAULT_CONFIG_FILE_NAME = "postchi"

func getConfigReader(configFileName string) (io.Reader, error) {
	_, err := os.Stat(configFileName)
	if err == nil {
		return os.Open(configFileName)
	}
	configFileName = DEFAULT_CONFIG_FILE_NAME + ".yaml"
	_, err = os.Stat(configFileName)
	if err == nil {
		return os.Open(configFileName)
	}

	configFileName = DEFAULT_CONFIG_FILE_NAME + ".yml"
	_, err = os.Stat(configFileName)
	if err == nil {
		return os.Open(configFileName)
	}
	return nil, errors.New("could not find any config file")

}

func (s *Context) formatString(str string) string {
	t, err := template.New("str").Parse(str)
	if err != nil {
		log.Fatalln(err)
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, s.ProcessedVars)
	if err != nil {
		log.Fatalln(err)
	}

	return buff.String()
}

func (r *request) toHttpRequest(state Context) *http.Request {
	route := state.formatString(r.Route)
	method := http.MethodGet
	if r.Method != "" {
		method = r.Method
	}
	var body io.Reader
	if r.Body != "" {
		r.Body = state.formatString(r.Body)
		body = strings.NewReader(r.Body)
	}
	req, err := http.NewRequest(method, route, body)
	if err != nil {
		log.Fatalln(err)
	}
	for key, v := range state.Defaults.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	for key, v := range r.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	queries := map[string]string{}

	for key, v := range state.Defaults.Query {
		queries[key] = state.formatString(v)
	}

	for key, v := range r.Query {
		queries[key] = state.formatString(v)
	}

	var queriesString []string
	for key, v := range queries {
		queriesString = append(queriesString, fmt.Sprintf("%s=%s", key, state.formatString(v)))
	}

	if queriesString != nil {
		req.URL.RawQuery = strings.Join(queriesString, "&")
	}

	return req
}

func verboseFormatRequest(req *http.Request) string {
	base := fmt.Sprintf("%s %s", req.Method, req.URL.String())
	for k, v := range req.Header {
		base += fmt.Sprintf("\n%s: %s", k, v)
	}

	base += "\n"

	return base
}

func verboseFormatResponse(resp *http.Response) string {
	base := resp.Status + "\n"
	for k, v := range resp.Header {
		base += fmt.Sprintf("\n%s: %s", k, v)
	}

	base += "\n"
	return base

}

func interactive() (*http.Response, error) {
	readFromEditor := func(editor string) (string, error) {
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}

		fd, err := os.CreateTemp("", "postchi-req")
		if err != nil {
			return "", err
		}
		defer os.Remove(fd.Name())

		cmd := exec.Command("sh", "-c", editor+" "+fd.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return "", err
		}
		b, err := ioutil.ReadFile(fd.Name())
		if err != nil {
			return "", err
		}

		return string(b), nil
	}
	reqText, err := readFromEditor("")
	if err != nil {
		return nil, err
	}

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reqText)))
	if err != nil {
		return nil, err
	}

	if req.RequestURI != "" {
		req.RequestURI = ""
	}

	client := http.Client{}
	return client.Do(req)
}

func main() {
	var requestName string
	var requestsFile string
	var interactiveMode bool
	var openEditorWithRespone bool
	var verbose bool
	flag.StringVar(&requestsFile, "f", "", "request file, defaults to postchi.yaml")
	flag.BoolVar(&interactiveMode, "i", false, "interactive mode will open your EDITOR and you write your request in HTTP format")
	flag.BoolVar(&openEditorWithRespone, "e", true, "open your $EDITOR with the output")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.Parse()
	args := flag.Args()

	if len(args) > 0 {
		requestName = args[0]
	}

	if requestName == "" || interactiveMode {
		resp, err := interactive()
		if err != nil {
			log.Fatalln(err)
		}
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(string(body))
		return
	}

	if requestName == "" {
		log.Fatalln("you need to specify request name")
	}

	configReader, err := getConfigReader(requestsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	ctx, err := newContext(args[1:], yaml.NewDecoder(configReader).Decode)

	var client http.Client
	if req, exists := ctx.Requests[requestName]; exists {
		hReq := req.toHttpRequest(ctx)
		if verbose {
			fmt.Println(verboseFormatRequest(hReq))
			fmt.Println("----------------------------")
		}
		resp, err := client.Do(hReq)
		if err != nil {
			log.Fatalln(err.Error())
		}
		if resp.StatusCode > 299 || resp.StatusCode < 200 {
			log.Println("Status Code: ", resp.StatusCode)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		if verbose {
			fmt.Println(verboseFormatResponse(resp))
			fmt.Println("++++++++++++++++++++++++++")
		}
		fmt.Fprint(os.Stdout, string(body))
	}
}
