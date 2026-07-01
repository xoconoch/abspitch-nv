module nmft-plugin

go 1.25

require github.com/navidrome/navidrome/plugins/pdk/go v0.0.0-20260701232447-b405252f51c6

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/extism/go-pdk v1.1.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/navidrome/navidrome => ../navidrome/

// Note: Adjust the right side of the replace directive to point to your local Navidrome source code checkout
