# Proces Verbal de Transare

Aplicație desktop pentru completarea, arhivarea și tipărirea procesului verbal
de transare folosit de S.C. Largiana Carn S.R.L.

## Cerințe

- Go 1.25+
- Node 20+
- Wails CLI v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Dezvoltare

```bash
wails dev
```

## Teste

```bash
go test ./...
cd frontend && npm test
```

## Build

```bash
wails build                          # macOS
wails build -platform windows/amd64  # Windows, de pe macOS
```

Ambele dependențe native (`modernc.org/sqlite`, `github.com/go-pdf/fpdf`) sunt
pur Go, deci cross-compilarea nu are nevoie de un toolchain Windows.

## Date

Baza de date SQLite se creează la prima pornire în directorul de configurare al
utilizatorului:

- macOS: `~/Library/Application Support/proces-verbal-transare/data.db`
- Windows: `%AppData%\proces-verbal-transare\data.db`

La prima pornire se populează lista celor 19 produse de pe formularul tipărit;
poate fi modificată din ecranul **Setări**.
