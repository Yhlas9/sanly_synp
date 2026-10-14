# Sanly Synp

**Mugallymlaryň işini ýeňilleşdirýän uniwersal oflaýn okuw platformasy.**

Matematika, fizika, himiýa, biologiýa we informatika dersleri üçin bitewi sanly
platforma: sapagy taýýarlamak, düşündirmek, bilimi barlamak we netijeleri seljermek.

Platforma **internetsiz** işleýär. Mugallym ony öz kompýuterinde işe girizýär,
okuwçylar bolsa şol bir Wi-Fi tora ýa-da hotspot-a birigip, brauzer arkaly goşulýarlar.
Goşmaça programma gurnamak gerek däl — telefondaky brauzer ýeterlik.

---

## Mazmuny

1. [Çalt başlangyç](#çalt-başlangyç)
2. [Talaplar](#talaplar)
3. [Ulanyjylar we sahypalar](#ulanyjylar-we-sahypalar)
4. [Gollanma: ulgam dolandyryjysy](#gollanma-ulgam-dolandyryjysy)
5. [Gollanma: mugallym](#gollanma-mugallym)
6. [Gollanma: okuwçy](#gollanma-okuwçy)
7. [Mümkinçilikleriň doly sanawy](#mümkinçilikleriň-doly-sanawy)
8. [Görkezmeden öňki barlag](#görkezmeden-öňki-barlag)
9. [Ätiýaçlyk nusga we dikeltmek](#ätiýaçlyk-nusga-we-dikeltmek)
10. [Tehniki maglumat](#tehniki-maglumat)
11. [Öňünden taýýarlanan bölekler](#öňünden-taýýarlanan-bölekler)
12. [Kynçylyklar we çözgütleri](#kynçylyklar-we-çözgütleri)

---

## Çalt başlangyç

```bash
./start.sh
```

Programma işe girenden soň ekranda salgylar görkezilýär:

```
  Mugallym:  http://localhost:8080/

  Okuwçylar üçin salgylar (şol bir Wi-Fi ýa-da hotspot):
     http://10.12.2.83:8080/okuwcy

  Giriş:  mugallym / mugallym
```

Başlangyç hasaplar:

| Hasap | Login | Açar söz | Nirä girýär |
|---|---|---|---|
| Mugallym | `mugallym` | `mugallym` | `/` |
| Ulgam dolandyryjysy | `admin` | `admin` | `/dolandyrys` |

> **Ilkinji işiňiz: açar sözleri çalşyň.**
> Mugallym — `/mugallym` sahypasynda, dolandyryjy — `/dolandyrys` → Mugallymlar.

### Işe girizmegiň sazlamalary

Ähli sazlamalar gurşaw üýtgeýjileri arkaly berilýär, hemmesiniň başlangyç bahasy bar:

| Üýtgeýji | Başlangyç bahasy | Näme üçin |
|---|---|---|
| `DATABASE_URL` | `postgres://localhost:5432/sanly_synp?sslmode=disable` | Maglumat binýadyna birikme |
| `ADDR` | `:8080` | Serweriň diňleýän porty |
| `WEB_DIR` | `web` | Sahypalaryň bukjasy |
| `MIGRATIONS_DIR` | `migrations` | Shemanyň faýllary |
| `SEED_FILE` | `seed/content.sql` | Nusgalyk mazmun |
| `UPLOAD_DIR` | `uploads` | Amaly işleriň faýllary |

Mysal: başga portda işe girizmek —

```bash
ADDR=:9000 ./start.sh
```

---

## Talaplar

- **Go 1.25** ýa-da täzesi — programmany ýygnamak üçin;
- **PostgreSQL 16** — ýerli kompýuterde işleýän bolmaly.

Başga hiç zat gerek däl. Şriftler, kitaphanalar, interaktiw modeller we QR
döredijisi programma bilen bilelikde **ýerli görnüşde** elýeterlidir —
internetden hiç zat ýüklenmeýär.

Baglylyklar `vendor/` bukjasynda saklanýar, şonuň üçin internetsiz hem
ýygnap bolýar:

```bash
GOFLAGS=-mod=vendor GOPROXY=off go build -o sanly-synp ./cmd/server
```

---

## Ulanyjylar we sahypalar

| Sahypa | Kim üçin | Näme edilýär |
|---|---|---|
| `/` | Mugallym | Sapagy başlatmak, ekrany görkezmek, synag, netijeler |
| `/materiallar` | Mugallym | Sapaklar, soraglar, synaglar, toparlar, žurnal |
| `/mugallym` | Mugallym | Öz otagy: materiallaryň sanlary, geçirilen sapaklar, açar sözi çalyşmak |
| `/hasabat` | Mugallym | Aýratyn okuwçynyň çap edilýän hasabaty |
| `/dolandyrys` | Dolandyryjy | Hasaplar, okuwçylar, ätiýaçlyk nusgalar, ulgam |
| `/okuwcy` | Okuwçy | Sapaga goşulmak we işlemek |
| `/profil` | Okuwçy | Şahsy otag: bahalaryň taryhy |

### Ygtyýarlyklar

Ygtyýarlyklar **serwer tarapyndan** barlanýar — diňe düwmäni gizlemek bilen çäklenilmeýär:

- okuwçy başga okuwçynyň jogaplaryny, netijesini ýa-da mugallymyň panelini açyp bilmeýär;
- mugallym diňe **öz** materiallaryny görýär we üýtgedýär;
- dolandyryş bölümine diňe `admin` hukukly hasap girýär.

---

## Gollanma: ulgam dolandyryjysy

`/dolandyrys` sahypasy, dört bölümden ybarat.

### 1. Umumy syn

Maglumatlaryň möçberi, okuwçylar üçin ýerli salgylar we serweriň işleýän wagty.

### 2. Mugallymlar

**Täze mugallym döretmek:**

1. «Täze hasap» bölümine login we doly ady ýazyň.
2. Açar sözi **boş goýuň** — ulgam özi döreder.
3. Hukugyny saýlaň: mugallym ýa-da ulgam dolandyryjysy.
4. «Hasap döretmek» düwmesine basyň.

Login we açar söz çap edilýän tablisada **bir gezek** görkezilýär. Çap edip beriň.

Sanawda her mugallymyň näçe sapagy, geçirilişi, topary we okuwçysy bardygy görünýär.

> Soňky dolandyryjyny we öz hasabyňyzy pozup bolmaýar — ulgama girer ýaly adam galmaly.

### 3. Okuwçylar

**Ähli toparlar** — haýsy mugallyma degişlidigi, näçe okuwçy bardygy we näçesiniň
hasabynyň bardygy (`2 / 3`).

**Topara hasap bermek:** toparyň hatarynda «Hasap döretmek» düwmesine basyň.
Hasaby bolmadyk ähli okuwçylara login we açar söz berilýär, sanaw çap edilýär.

**Ähli okuwçylar** — ady, topary, mugallymy, logini, näçe iş tabşyrandygy we
soňky girişi. Gözlemek üçin topar boýunça süzgüç bar.

**Açar sözi ýitse:** okuwçynyň hatarynda «Täze açar söz». Köne açar söz şol
pursatda işlemegini bes edýär.

### 4. Ätiýaçlyk nusgalar

«Täze nusga döretmek» — maglumat binýadynyň nusgasyny döredýär.
«Ýükle» — nusgany kompýutere göçürýär. «Dikelt» — maglumatlary şol nusga dolaýar.

> Dikeltmek **yzyna gaýtaryp bolmaýan** amal: şondan soňky ähli işler ýitýär.
> Şonuň üçin ol `DIKELT` sözüni ýazmak bilen tassyklanýar.

### 5. Ulgam

Tehniki maglumatlar we howpsuzlyk barlagy.

---

## Gollanma: mugallym

### Ädim 1. Topar döretmek we sanaw girizmek

1. `/materiallar` → **Toparlar**.
2. Toparyň adyny ýazyp «Goşmak» (mysal: `9-A`).
3. Toparyň hatarynda «Sanaw» düwmesi.
4. Okuwçylaryň atlaryny **her setire bir at** görnüşinde ýazyp «Sanawa goşmak».

> Bir mugallymda şol bir atly iki topar bolup bilmeýär — bolsa žurnal ikä bölünerdi.

### Ädim 2. Okuwçylara hasap bermek (islege görä)

«Hasap döretmek» düwmesi hasapsyz okuwçylara login we açar söz berýär.
Login atdan awtomatik döredilýär: «Aýgül Nurýewa» → `aygul.nuryewa`.

Sanaw çap edilip paýlanýar. Hasap bilen okuwçy `/profil` sahypasynda öz
bahalarynyň taryhyny görýär.

> Hasap hökmany däl: kod bilen goşulmak hemişe işleýär.

### Ädim 3. Sapak taýýarlamak

`/materiallar` → **Sapaklar**.

**Iň çalt ýol — nusgalyk sapakdan başlamak.** Platforma bilen bäş ders gelýär,
olar `nusgalyk` diýlip bellenen. «Nusga almak» düwmesi şol sapagyň ähli
soraglary bilen bilelikde **öz nusgaňyzy** döredýär we derrew açýar —
soňra islendik zady üýtgedip bilersiňiz.

**Boş sahypadan başlamak:** «Täze sapak» → ders, at, mowzuk, okuw maksady,
düşündiriş teksti, özbaşdak ýumuş.

**Interaktiw görkezme** saýlanýar we sazlamasy JSON görnüşinde berilýär.
Görkezmäniň nusgasy şol ýerde görünýär:

| Görkezme | Ders | Okuwçy näme edýär |
|---|---|---|
| `function_plot` | Matematika | a, b, c parametrlerini üýtgedip, parabolany, depe nokadyny we kökleri görýär |
| `circuit` | Fizika | U we R üýtgedip, tok güýjüni, kuwwaty we lampanyň ýagtylygyny görýär |
| `reaction` | Himiýa | Koeffisiýent saýlap, atomlaryň deňleşýändigini görýär |
| `cell` | Biologiýa | Öýjügiň bölegine basyp, onuň wezipesini okaýar |
| `algorithm` | Informatika | Tertipleşdirmäni ädimme-ädim yzarlaýar |

> Okuwçynyň hereketi netijä täsir etmeli. Statik surat interaktiw görkezme
> hasaplanmaýar.

### Ädim 4. Sorag goşmak

Sapagyň aşagyndaky «Sorag goşmak». Ýedi görnüş bar:

| Görnüş | Nähili barlanýar |
|---|---|
| Jogap saýlamak | Awtomatik, bir ýa-da birnäçe dogry jogap |
| Gysga jogap | Awtomatik, kabul edilýän jogaplaryň sanawy boýunça |
| Sanly jogap | Awtomatik, ölçeg birligi we ýalňyşlyk çägi bilen |
| Gabat getirmek | Awtomatik, **bölekleýin bal** berilýär |
| Yzygiderlilik düzmek | Awtomatik, **bölekleýin bal** berilýär |
| Açyk jogap | Mugallym el bilen bahalandyrýar |
| Amaly iş | Mugallym el bilen; okuwçy **faýl goşup** bilýär |

Her soraga bal, görkezme (hint) we düşündiriş goşup bolýar. Düşündiriş
netijede we hasabatda görkezilýär.

### Ädim 5. Synag ýygnamak

`/materiallar` → **Synaglar** → «Täze synag».

Dersi saýlanyňyzda şol dersiň soraglar banky açylýar — gerekli soraglary belläň.

Sazlamalar:

- **dowamlylygy** (minut);
- **soraglary garyşdyrmak** — her okuwça başga tertipde berilýär;
- **warýantlary garyşdyrmak**;
- **gaýtadan tabşyrmaga rugsat** — bolmasa, tabşyrylan işi üýtgedip bolmaýar;
- **netijeler haçan açylýar**.

> Synaga diňe öz soraglaryňyzy ýa-da nusgalyk soraglary goşup bolýar.

### Ädim 6. Sapagy geçirmek

Baş sahypada (`/`) sapagy we synagy saýlaň, topary görkeziň, görnüşini saýlaň:

| Görnüş | Nähili işleýär |
|---|---|
| **Janly sapak** | Mugallym synagy özi başladýar, ähli okuwçy bir wagtda işleýär |
| **Öý işi** | Tabşyryk bellenen möhlete çenli açyk. Okuwçy özüne amatly wagtda girýär; möhleti **işe başlan pursatyndan** hasaplanýar, ýöne ýapylýan wagtyndan geçmeýär |

«Sapagy başlat» — ekranda **5 harply kod**, ýerli toruň salgysy we **QR kod**
peýda bolýar. Okuwçy kamerany tutsa, göni sapaga düşýär.

Panelde elýeterli:

- **Sapagy görkezmek** — okuwçylarda sapagyň mazmuny we interaktiw görkezme açylýar;
- **Synagy başlatmak** — möhlet başlaýar (janly sapakda);
- **Goşulmagy ýapmak** — täze gatnaşyjylar kabul edilmeýär;
- **Ekrany görkezmek** — siziň ekranyňyz okuwçylaryň enjamlarynda;
- **+5 minut goşmak** — möhleti uzaltmak;
- **Netijeleri okuwçylara açmak**.

Gapdalda kim goşuldy, kim näçe jogap berdi we kim tabşyrdy — janly görünýär.

### Ädim 7. Netijeleri seljermek

«Netijeleri görmek» — tablisa: okuwçy, bal, göterim, ýagdaý, el bilen
barlanmaga garaşýan ýumuşlaryň sany.

El bilen barlanmadyk ýumuş bar bolsa netije **deslapky** diýlip bellenýär.

Tablisanyň aşagynda **«Gaýtadan düşündirmeli mowzuklar»** — topar boýunça iň
köp ýalňyşylan soraglar. Diňe umumy bal ýeterlik däl: şu sanaw ertirki sapagy
nämeden başlamalydygyny görkezýär.

- **«Işi açmak»** — okuwçynyň her jogabyny görmek we el bilen baha goýmak.
  Baly düzedeniňizde öňki bal, täze bal we sebäbi saklanýar.
- **«Hasabat»** — çap etmäge amatly resminama, mugallymyň goly üçin ýeri bar.
- **CSV ýükle** — tablisany Excel üçin.

### Ädim 8. Žurnal

`/materiallar` → **Žurnal** → topary saýlaň.

Işler keseligine, okuwçylar dikligine, her öýjükde göterim, aşagynda ortaça baha.
Bir sapagyň içinde görünmeýän zady görkezýär — okuwçynyň wagtyň dowamyndaky ösüşini.

### Öz otagyňyz

`/mugallym` — taýýarlan materiallaryňyz sanlarda, geçiren sapaklaryňyzyň taryhy
(näçe okuwçy goşuldy, näçesi tabşyrdy, toparyň ortaça netijesi) we **öz açar
sözüňizi çalyşmak**.

---

## Gollanma: okuwçy

### Sapaga goşulmak

1. Telefonda brauzeri açyp, mugallymyň aýdan salgysyna giriň
   (mysal: `http://10.12.2.83:8080/okuwcy`) ýa-da **QR kody okaň**.
2. **Sapak kodyny** (5 harp) giriziň.
3. Adyňyzy ýazyň. Sapak topara berkidilen bolsa — sanawdan saýlaň.

### Işlemek

- Sapak görkezilende mazmuny we **interaktiw görkezmäni** synap bilersiňiz.
- Mugallym ekranyny görkezende ol sahypada peýda bolýar, «Ulaltmak» bilen ulaldyp bolýar.
- Synagda her jogabyň aşagynda ýagdaýy görünýär:
  **saklandy** · **iberilmegine garaşylýar** · **jogap ýok**.
- Amaly işde netijäni **faýl görnüşinde** hem goşup bolýar (surat, PDF, tablisa; 10 MB çenli).

> **Tor kesilse alada etmäň.** Jogaplaryňyz enjamyňyzda saklanýar we birikme
> dikeldilende awtomatik ugradylýar. Sahypany ýapmaň.

### Şahsy otag

Mugallym hasap beren bolsa, `/profil` salgysyndan login we açar söz bilen giriň.
Ol ýerde ähli işleriňiziň taryhy, her biriniň baly we ortaça netijäňiz bar.

> Netije diňe mugallym ony açandan soň görünýär.

---

## Mümkinçilikleriň doly sanawy

### Ygtybarlylyk

- **Möhleti serwer hasaplaýar.** Enjamdaky sagady üýtgetmek synaga täsir etmeýär.
- **Dogry jogaplaryň açary** synag dowam edýärkä okuwçynyň enjamyna **ugradylmaýar**.
- **Tor kesilende** jogaplar ýerli nobatda saklanýar; `client_key` arkaly şol bir
  jogap iki gezek hasaba alynmaýar.
- **Tabşyrylan iş** gaýtadan tabşyrmaga rugsat berilmedik bolsa üýtgedilmeýär.
- **Synagyň mazmuny** başlanan pursatynda doňdurylýar: soraglar bankyndaky soňky
  düzedişler öň tabşyrylan işleriň netijesini üýtgetmeýär.
- **Faýly** diňe sapagyň mugallymy we ony ugradan okuwçy görüp bilýär.

### Ekrany görkezmek

Mugallymyň ekrany ýa-da saýlanan penjiresi okuwçylaryň enjamlarynda göni görünýär.
Kadrlar JPEG görnüşinde WebSocket arkaly ugradylýar: **720p, 5 kadr/sekunt**.
Okuwçy şekili ulaldyp bilýär; görkezmek bes edilende sapak sahypasyna dolanýar.

### Ölçelen netijeler

30 okuwçynyň bir wagtda işlemegi synag edildi (600 jogap):

| Görkeziji | Talap | Ölçelen |
|---|---|---|
| Jogaby saklamak | ≤ 2 sek | ortaça **8 ms**, 95% ýagdaýda **14 ms** |
| Netijäni çykarmak (30 × 20) | ≤ 5 sek | **2 ms** |
| 30 okuwçynyň goşulmagy | ≤ 1 min | **0,07 sek** |

---

## Görkezmeden öňki barlag

```bash
./check.sh
```

Bir buýruk bilen geçirilýän barlaglar: PostgreSQL işleýärmi, programma internetsiz
ýygnalýarmy, sahypalar açylýarmy, `web/` içinde daşarky salgy barmy, ýerli toruň
salgysy barmy, bäş ders boýunça mazmun ýerindemi, ýedi görnüşli ýumuş barmy,
başlangyç açar sözler çalşyldymy, soňky ätiýaçlyk nusga haçan alnan.

### El bilen barlamaly

Skript bulary barlap bilmeýär:

1. **Mugallymyň paneli `localhost` arkaly açylmaly.** Brauzer ekrany görkezmäge
   diňe howpsuz çeşmede rugsat berýär, `localhost` şol sana girýär. Ýerli IP
   arkaly açsaňyz, ekrany görkezmek işlemeýär.
   *Okuwçylar üçin bu çäklendirme ýok.*
2. Telefonda ýerli salgy açylýarmy we sapaga 1 minutda goşulyp bolýarmy.
3. Interneti öçüreniňde platforma işlemegini dowam edýärmi.
4. Telefonda Wi-Fi öçürip-ýakanyňda jogap ýitmeýärmi.
5. Proýektorda tekst okalýarmy.

---

## Ätiýaçlyk nusga we dikeltmek

```bash
./backup.sh                               # nusga döretmek
./restore.sh backups/sanly-synp-….sql     # dikeltmek
```

Nusga maglumat binýadyny **we** amaly işleriň faýllaryny (`uploads/`) öz içine alýar —
faýllar aýratyn `.tar.gz` görnüşinde şol bir at bilen saklanýar.

Şol bir işi `/dolandyrys` → Ätiýaçlyk nusgalar bölüminden hem edip bolýar.

### Synag maglumatlaryny arassalamak

```bash
./reset-demo.sh
```

Geçirilen sapaklary, toparlary we synag üçin döredilen materiallary pozýar.
Nusgalyk bäş ders ýerinde galýar.

---

## Tehniki maglumat

| Bölek | Çözgüt | Näme üçin |
|---|---|---|
| Serwer | **Go 1.25**, standart `net/http` | Bir ýygnalan faýl: goşmaça gurnama gerek däl |
| Maglumat binýady | **PostgreSQL 16**, arassa SQL | Ygtybarly saklanyş, tranzaksiýalar; ORM ulanylmadyk |
| Janly birikme | **WebSocket** (`gorilla/websocket`) | Ekran görkezmek we sapagyň janly ýagdaýy |
| Interfeýs | **Vue 3.5**, ýerli faýl, ýygnaýjysyz | Brauzer göni açýar |
| Şriftler | Ulgamyň öz şriftleri | Internetden hiç zat ýüklenmeýär |

### Gurluş

```
cmd/server/        — programmanyň başlangyç nokady
internal/store/    — maglumat binýady, arassa SQL; awtomatik barlag
internal/api/      — HTTP we WebSocket
internal/hub/      — sapagyň janly ýagdaýy we ekran görkezmek
migrations/        — shema (tertip boýunça ýerine ýetirilýär)
seed/content.sql   — bäş ders boýunça nusgalyk mazmun
web/               — ähli sahypalar we ýerli kitaphanalar
uploads/           — amaly işleriň faýllary
backups/           — ätiýaçlyk nusgalar
```

### Buýruklar

| Buýruk | Näme edýär |
|---|---|
| `./start.sh` | Işe girizmek (gerek bolsa binýady döredýär) |
| `./check.sh` | Görkezmeden öňki barlag |
| `./backup.sh` | Ätiýaçlyk nusga |
| `./restore.sh <faýl>` | Nusgadan dikeltmek |
| `./reset-demo.sh` | Synag maglumatlaryny arassalamak |

---

## Öňünden taýýarlanan bölekler

Bäsleşigiň düzgünnamasynyň **V.7** bendine laýyklykda ulanylan açyk çeşmeli
kitaphanalar:

| Bölek | Wersiýa | Ygtyýarnama |
|---|---|---|
| `jackc/pgx` | v5.11 | MIT |
| `gorilla/websocket` | v1.5 | BSD-2 |
| `golang.org/x/crypto` (bcrypt) | v0.58 | BSD-3 |
| Vue | 3.5 | MIT |
| `qrcode-generator` (Kazuhiko Arase) | 1.4 | MIT |

Platformanyň özeni — maglumat shemasy, awtomatik barlag, sapak dolandyryşy,
ekran görkezmek, nobat arkaly ygtybarly jogap saklaýyş, interaktiw görkezmeler,
žurnal we hasabatlar — bäsleşigiň dowamynda ýazyldy.

---

## Kynçylyklar we çözgütleri

**«Ekrany görkezmek» işlemeýär.**
Mugallymyň paneli `localhost` arkaly açylmaly. Brauzer ekrany paýlaşmaga diňe
howpsuz çeşmede rugsat berýär.

**Okuwçy salga girip bilmeýär.**
Telefon şol bir Wi-Fi tora ýa-da hotspot-a birigen bolmaly. Salgyny `/api/net`
ýa-da serweriň başlangyç ekrany görkezýär. Käbir Wi-Fi nokatlary enjamlaryň
biri-birini görmegini gadagan edýär (client isolation) — şonda hotspot ulanyň.

**Serwer işe girmeýär: `maglumat binýady`.**
PostgreSQL işlemeýär. `brew services start postgresql@16` ýa-da `pg_isready` bilen barlaň.

**Mugallym gaýtadan girmeli boldy.**
Giriş sessiýalary ýatda saklanýar, serwer gaýtadan işe girizilende olar ýitýär.
Okuw maglumatlary bu ýagdaýda ýitmeýär — olar binýatda.

**Okuwçy açar sözüni ýitirdi.**
Mugallym: `/materiallar` → Toparlar → Sanaw → «Täze açar söz».
Dolandyryjy: `/dolandyrys` → Okuwçylar.

**Sapagy topara berkitdim, okuwçy girip bilmeýär.**
Sapak topara berkidilende diňe sanawdaky okuwçylar girip bilýär. Ady sanawa
goşuň ýa-da topary saýlaman sapak geçiriň.

**Nusgalyk sapagy üýtgedip bolmaýar.**
Olar diňe okamak üçin. «Nusga almak» düwmesi öz nusgaňyzy döredýär — ony
islendik görnüşde üýtgedip bilersiňiz.
