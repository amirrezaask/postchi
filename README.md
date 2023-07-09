# postchi
Yet another HTTP client that doesn't SUCK.



## Sample config file
```yaml
env:
  baseURL:
    source: process 
    name: BASE_URL
    value: http://localhost:1323/api/v2/asset/167
  randomenv:
    source: plain
    value: secret token
  jwtToken:
    source: process
    name: JWT_TOKEN
    value: ""
  assetID:
    source: cli
    arg_number: 1

defaults:
  headers:
    Authorization: "Bearer {{ .jwtToken }}"
    Content-Type: "application/json"
    Accepts: "application/json"

requests:
  index:
    route: "{{ .baseURL }}"
  feeds:
    route: "{{ .baseURL }}/feed"
```

## Usage
```bash
postchi -file postchi.yaml -name feeds
postchi -name feeds #defaults to using postchi.yaml in your current PWD
```
