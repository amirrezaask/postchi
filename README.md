# postchi
Yet another HTTP client that doesn't SUCK.



## Sample config file
```yaml
vars:
  baseURL:
    source: process 
    name: BASE_URL
    value: https://postman-echo.com
  jwtToken:
    source: process
    name: JWT_TOKEN
    value: "secret token"

defaults:
  headers:
    Authorization: "Bearer {{ .jwtToken }}"
    Content-Type: "application/json"
    Accepts: "application/json"

requests:
  postman-echo:
    route: "{{ .baseURL }}/get"
    body: >
      { "name": "amirreza" }
 
```

## Usage
```bash
postchi -f postchi.yaml feeds
postchi feeds # defaults to using postchi.yaml in your current PWD
```
I recommend having `jq` installed on your system and use this tool in combination of that and a text editor, so you can analyze output of the api more easily.
```bash
postchi index someid | jq | code -
postchi index someid | jq | vim
```
### Interactive mode
```bash
postchi -i # will open up your $EDITOR and you write your request in HTTP format
```
