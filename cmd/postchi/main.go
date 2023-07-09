package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

type envSource string

const (
	envSource_Plain   = "plain"
	envSource_Cli     = "cli"
	envSource_Process = "process"
)

type envConfig struct {
	Source    envSource `json:"source" yaml:"source"`
	Name      string    `json:"name" yaml:"name"`
	ArgNumber int       `json:"arg_number" yaml:"arg_number"`
	Value     string    `json:"value" yaml:"value"`
}
type requestConfig struct {
	Method  string
	Route   string            `json:"route" yaml:"route"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Body    string            `json:"body" yaml:"body"`
}

type config struct {
	Env      map[string]envConfig     `json:"env" yaml:"env"`
	Defaults requestConfig            `json:"defaults" yaml:"defaults"`
	Requests map[string]requestConfig `json:"requests" yaml:"requests"`
}

type state struct {
	env map[string]string
	cfg config
}

func newState(args []string, cfg config) state {
	env := map[string]string{}
	for k, v := range cfg.Env {
		switch v.Source {
		case envSource_Process:
			envValue := os.Getenv(v.Name)
			if envValue == "" {
				envValue = v.Value
			}
			env[k] = envValue
		case envSource_Cli:
			if len(args) > v.ArgNumber && args[v.ArgNumber] != "" {
				env[k] = args[v.ArgNumber]
			} else {
				env[k] = v.Value
			}
		case envSource_Plain:
			env[k] = v.Value
		}
	}

	return state{
		env: env,
		cfg: cfg,
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
		panic(err)
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, s.env)
	if err != nil {
		panic(err)
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
		panic(err)
	}
	for key, v := range state.cfg.Defaults.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	for key, v := range r.Headers {
		req.Header.Add(key, state.formatString(v))
	}

	return req
}

func main() {
	var requestName string
	var requestsFile string
	flag.StringVar(&requestName, "name", "", "request you want to send")
	flag.StringVar(&requestsFile, "file", "", "request file, defaults to postchi.yaml")
	flag.Parse()

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
			panic(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Fprint(os.Stdout, string(body))
	}
}
