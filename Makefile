# Instalatoare pentru Proces Verbal de Transare.
#
#   make mac       -> dist/ProcesVerbalTransare-<versiune>-macOS.dmg
#   make windows   -> dist/ProcesVerbalTransare-<versiune>-Windows-Setup.exe
#   make all       -> ambele
#
# Versiunea are o singura sursa de adevar: campul info.productVersion din wails.json.

VERSION      := $(shell python3 -c "import json;print(json.load(open('wails.json'))['info']['productVersion'])")
PRODUCT_NAME := Proces Verbal de Transare
DIST         := dist
APP_BUNDLE   := build/bin/$(PRODUCT_NAME).app
DMG          := $(DIST)/ProcesVerbalTransare-$(VERSION)-macOS.dmg
NSIS_OUT     := build/bin/proces-verbal-transare-amd64-installer.exe
SETUP_EXE    := $(DIST)/ProcesVerbalTransare-$(VERSION)-Windows-Setup.exe

.PHONY: all mac windows clean check-wails check-nsis version

all: mac windows

version:
	@echo $(VERSION)

check-wails:
	@command -v wails >/dev/null 2>&1 || { \
		echo "Eroare: 'wails' nu este instalat."; \
		echo "  go install github.com/wailsapp/wails/v2/cmd/wails@latest"; \
		exit 1; }

check-nsis:
	@command -v makensis >/dev/null 2>&1 || { \
		echo "Eroare: 'makensis' (NSIS) nu este instalat - necesar pentru instalatorul Windows."; \
		echo "  brew install nsis"; \
		exit 1; }

## macOS: bundle universal (Intel + Apple Silicon) impachetat intr-un .dmg
mac: check-wails
	wails build -platform darwin/universal -clean
	@rm -rf "$(APP_BUNDLE)"
	mv "build/bin/proces-verbal-transare.app" "$(APP_BUNDLE)"
	@mkdir -p $(DIST)
	build/darwin/make-dmg.sh "$(APP_BUNDLE)" "$(DMG)" "$(PRODUCT_NAME)"
	@echo "Gata: $(DMG)"

## Windows: executabil amd64 + instalator NSIS per-utilizator
##
## NU adauga "-installscope user": scopul este fixat in
## build/windows/installer/project.nsi prin REQUEST_EXECUTION_LEVEL, iar flagul
## CLI ar redefini aceeasi constanta si ar opri compilarea. Din acest motiv
## wails raporteaza "Install Scope | machine" - raportul este gresit, project.nsi
## are ultimul cuvant (verificat: manifestul instalatorului contine "asInvoker").
windows: check-wails check-nsis
	wails build -platform windows/amd64 -clean -nsis
	@mkdir -p $(DIST)
	cp "$(NSIS_OUT)" "$(SETUP_EXE)"
	@echo "Gata: $(SETUP_EXE)"

clean:
	rm -rf $(DIST) build/bin
