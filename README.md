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

## Instalatoare

Ambele instalatoare se construiesc de pe macOS:

```bash
make mac        # dist/ProcesVerbalTransare-1.0.0-macOS.dmg
make windows    # dist/ProcesVerbalTransare-1.0.0-Windows-Setup.exe
make all        # ambele
```

Versiunea are o singură sursă: câmpul `info.productVersion` din `wails.json`.
Numele fișierelor rezultate o preiau automat.

Pentru instalatorul Windows este nevoie de NSIS:

```bash
brew install nsis
```

`make mac` produce un bundle universal (Intel + Apple Silicon) împachetat într-un
`.dmg` cu legătură către *Applications*. `make windows` produce un instalator
care se instalează **pentru utilizatorul curent**, în
`%LocalAppData%\Programs\Proces Verbal de Transare`, fără prompt UAC și fără
drepturi de administrator; adaugă o scurtătură în meniul Start și o intrare în
*Adăugare sau eliminare programe*.

### Semnare digitală

Aplicația **nu este semnată**. Instalatoarele funcționează, dar sistemul de
operare afișează un avertisment la prima pornire (vezi *Instalare* mai jos).

Dacă în viitor se achiziționează certificate, semnarea se activează prin
variabile de mediu, fără modificări de cod:

```bash
export APPLE_SIGNING_IDENTITY="Developer ID Application: ..."
export APPLE_NOTARY_PROFILE="numele-profilului-notarytool"
make mac
```

Pentru Windows, decomentează liniile `!finalize` / `!uninstfinalize` cu
`signtool` din `build/windows/installer/project.nsi`.

## Instalare

### macOS

1. Deschide fișierul `.dmg` și trage aplicația peste folderul *Applications*.
2. La prima pornire macOS o poate refuza („dezvoltator neidentificat” sau
   „aplicația este deteriorată”), pentru că nu este semnată. Se rezolvă o
   singură dată, din Terminal:

   ```bash
   xattr -dr com.apple.quarantine "/Applications/Proces Verbal de Transare.app"
   ```

Aceleași instrucțiuni se găsesc în fișierul `CITEȘTE-MĂ.txt` din interiorul
imaginii `.dmg`.

### Windows

1. Rulează `ProcesVerbalTransare-<versiune>-Windows-Setup.exe`.
2. Windows SmartScreen va afișa „Windows a protejat PC-ul”, pentru că
   instalatorul nu este semnat. Apasă **Informații suplimentare** →
   **Executare oricum**.
3. Instalarea decurge fără cerere de parolă de administrator.

Dezinstalarea se face din *Adăugare sau eliminare programe*. **Baza de date nu
este ștearsă la dezinstalare** — rămâne în `%AppData%\proces-verbal-transare`.

## Date

Baza de date SQLite se creează la prima pornire în directorul de configurare al
utilizatorului:

- macOS: `~/Library/Application Support/proces-verbal-transare/data.db`
- Windows: `%AppData%\proces-verbal-transare\data.db`

La prima pornire se creează două șabloane: *Carcasa Porc*, cu cele 19 produse
de pe formularul tipărit și un proces verbal de exemplu, și *Pulpa vita Angus*,
cu cele 5 produse ale ei. Un șablon este o listă "ce iese" cu numele ei; se pot
crea oricâte, din ecranul **Setări**, iar la crearea unui document se alege din
care șablon pornește. Numerotarea documentelor este comună tuturor șabloanelor.
