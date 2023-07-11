package main

import (
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
	"text/template"

	"github.com/amirrezaask/postchi/pkg/httpparser"
	"gopkg.in/yaml.v3"
)

type envSource string

const (
	envSource_Plain   = "plain"
	envSource_Args    = "args"
	envSource_Process = "process"
)

type varConfig struct {
	Source envSource `json:"source" yaml:"source"`
	Name   string    `json:"name" yaml:"name"`
	Index  int       `json:"index" yaml:"index"`
	Value  string    `json:"value" yaml:"value"`
}
type requestConfig struct {
	Method  string
	Route   string            `json:"route" yaml:"route"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Body    string            `json:"body" yaml:"body"`
}

type config struct {
	Vars     map[string]varConfig     `json:"vars" yaml:"vars"`
	Defaults requestConfig            `json:"defaults" yaml:"defaults"`
	Requests map[string]requestConfig `json:"requests" yaml:"requests"`
}

type state struct {
	vars map[string]string
	cfg  config
}

func newState(args []string, cfg config) state {
	vars := map[string]string{}
	for k, v := range cfg.Vars {
		switch v.Source {
		case envSource_Process:
			envValue := os.Getenv(v.Name)
			if envValue == "" {
				envValue = v.Value
			}
			vars[k] = envValue
		case envSource_Args:
			if len(args) > v.Index && args[v.Index] != "" {
				vars[k] = args[v.Index]
			} else {
				vars[k] = v.Value
			}
		case envSource_Plain:
			vars[k] = v.Value
		}
	}

	return state{
		vars: vars,
		cfg:  cfg,
	}
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

func (s *state) formatString(str string) string {
	t, err := template.New("str").Parse(str)
	if err != nil {
		log.Fatalln(err)
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, s.vars)
	if err != nil {
		log.Fatalln(err)
	}

	return buff.String()
}

func (r *requestConfig) toHttpRequest(state state) *http.Request {
	route := state.formatString(r.Route)
	method := http.MethodGet
	if r.Method != "" {
		method = r.Method
	}
	req, err := http.NewRequest(method, route, nil)
	if err != nil {
		log.Fatalln(err)
	}
	for key, v := range state.cfg.Defaults.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	for key, v := range r.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	return req
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

	req, err := httpparser.Parse(reqText)
	if err != nil {
		return nil, err
	}

	// spew.Dump(req)

	client := http.Client{}
	return client.Do(req)
}

func main() {
	var requestName string
	var requestsFile string
	var interactiveMode bool
	flag.StringVar(&requestName, "name", "", "request you want to send")
	flag.StringVar(&requestsFile, "file", "", "request file, defaults to postchi.yaml")
	flag.BoolVar(&interactiveMode, "interactive", false, "interactive mode will open your EDITOR and you write your request in HTTP format")
	flag.Parse()

	if interactiveMode {
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
	args := flag.Args()

	configReader, err := getConfigReader(requestsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	var cfg config
	err = yaml.NewDecoder(configReader).Decode(&cfg)
	if err != nil {
		log.Fatalln(err.Error())
	}

	state := newState(args, cfg)

	var client http.Client
	if req, exists := state.cfg.Requests[requestName]; exists {
		resp, err := client.Do(req.toHttpRequest(state))
		if err != nil {
			log.Fatalln(err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprint(os.Stdout, string(body))
	}
}
