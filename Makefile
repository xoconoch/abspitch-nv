.RECIPEPREFIX = >

PLUGIN_NAME := nmft-nv
PACKAGE_NAME := nmft-nv

.PHONY: build package clean

build:
> tinygo build -o $(PLUGIN_NAME).wasm -target=wasip1 -scheduler=none -buildmode=c-shared .

package: build
> cp $(PLUGIN_NAME).wasm plugin.wasm
> zip -j $(PACKAGE_NAME).ndp manifest.json plugin.wasm
> rm -f plugin.wasm $(PLUGIN_NAME).wasm

clean:
> rm -f $(PLUGIN_NAME).wasm $(PACKAGE_NAME).ndp
