# Sanly Synp — goramak üçin materiallar

> Düzgünnamanyň **VI.6** bendi tanyşdyrylyşda şu maglumatlary talap edýär:
> taslamanyň ady, maksady, çözülýän mesele, ulanylýan tehnologiýalar,
> esasy mümkinçilikler, bilim taýdan ähmiýeti, geljekde ösdürmegiň mümkinçilikleri.

---

## Çykyşyň başy (30 sekunt, ýatdan)

> Synp otagynda internet hemişe bolanok. Şonuň üçin mugallym sanly serişdelerden
> ýüz öwürýär: tagta, hek we kagyz test.
>
> **Sanly Synp** şol çäklendirmäni aýyrýar. Ol mugallymyň kompýuterinde işleýär
> we internet talap etmeýär. Okuwçylar öz telefonlary bilen mugallymyň Wi-Fi
> toruna birigip, sapaga bir minutda goşulýarlar.
>
> Häzir men internedi öçürýärin — we platforma işlemegini dowam etdirýär.
> Siz hem telefonyňyz bilen goşulyp bilersiňiz: kod ekranda.

---

## 1. Taslamanyň ady

**Sanly Synp** — mugallymlaryň işini ýeňilleşdirýän uniwersal oflaýn okuw platformasy.

## 2. Taslamanyň maksady

Matematika, fizika, himiýa, biologiýa we informatika mugallymlaryna sapagy
taýýarlamaga, düşündirmäge, okuwçylaryň bilimini barlamaga we netijeleri
seljermäge mümkinçilik berýän **bitewi** sanly platforma döretmek.

Esasy netije — mugallymyň gündelik işinde ulanyp biljek, hakykatdanam işleýän programma.

## 3. Çözülýän mesele

**Birinji mesele — internet.** Dünýädäki okuw platformalary internete baglydyr.
Internet bolmasa olar işlemeýär. Türkmenistanyň synp otaglarynda durnukly internet
hemişe elýeterli däl, şonuň üçin mugallym sanly gurallardan peýdalanyp bilmeýär.

**Ikinji mesele — enjam.** Her synpda kompýuter otagy ýok. Emma okuwçylaryň
aglabasynda telefon bar.

**Üçünji mesele — barlagyň zähmeti.** Mugallym 30 işi el bilen barlaýar we
netijede diňe baly görýär: synpyň nirede ýalňyşandygyny görmeýär.

**Biziň çözgüdimiz:** mugallymyň kompýuteri ýerli serwer bolýar. Okuwçylar
telefonlary bilen onuň Wi-Fi toruna ýa-da hotspot-a birigýärler. Internet gerek däl.

## 4. Ulanylýan tehnologiýalar

| Bölek | Çözgüt | Näme üçin |
|---|---|---|
| Serwer | **Go 1.25** | Bir ýygnalan faýl: goşmaça gurnama we ýükleme gerek däl |
| Maglumat binýady | **PostgreSQL 16**, arassa SQL | Ygtybarly saklanyş, tranzaksiýalar; ORM ulanylmadyk |
| Janly birikme | **WebSocket** | Ekran görkezmek we sapagyň janly ýagdaýy |
| Interfeýs | **Vue 3**, ýerli faýl | Ýygnaýjysyz: brauzer göni açýar |
| Şriftler | Ulgamyň şriftleri | Internetden hiç zat ýüklenmeýär |

## 5. Programmanyň esasy mümkinçilikleri

1. **Oflaýn iş.** Ähli esasy mümkinçilikler internetsiz işleýär.
2. **Bir minutda goşulmak.** Ekranda 5 harply kod, ýerli toruň salgysy we QR kod.
   Okuwçy kamerany tutup, göni sapaga düşýär.
3. **Ekrany görkezmek.** Mugallymyň ekrany okuwçylaryň telefonlarynda: 720p, 5 kadr/sekunt.
4. **Ýedi görnüşli ýumuş.** Jogap saýlamak, gysga jogap, sanly jogap, gabat getirmek,
   yzygiderlilik düzmek, açyk jogap, amaly iş.
5. **Tor kesilse-de jogap ýitmeýär.** Jogaplar ýerli nobatda saklanýar we birikme
   dikeldilende awtomatik ugradylýar; şol bir jogap iki gezek hasaba alynmaýar.
9. **Mugallym öz materialyny döredýär.** Sapak, interaktiw görkezme, sorag we synag
   platformanyň içinde döredilýär, nusgasy alynýar we üýtgedilýär.
10. **Synagyň sazlamalary.** Dowamlylyk, soraglaryň we warýantlaryň garyşdyrylmagy,
    gaýtadan tabşyrmaga rugsat, netijeleriň haçan açyljakdygy.
11. **Goşmaça wagt.** Mugallym synagyň möhletini uzaldyp, giç gelen işi kabul edip bilýär.
12. **Amaly işi faýl bilen tabşyrmak.** Okuwçy tejribäniň suratyny, tablisany ýa-da
    hasabat faýlyny goşup bilýär; faýly diňe mugallym we onuň özi görýär.
13. **Topar žurnaly.** Okuwçynyň işleri bir sapagyň içinde däl-de, toparyň
    žurnalynda ýygnanýar: ähli işler, her biriniň göterimi we ortaça baha.
    Mugallym diňe şu günki netijäni däl, ösüşiň taryhyny görýär.
14. **Ulgam dolandyryşy.** Mugallym hasaplaryny dolandyrmak, ätiýaçlyk nusga
    döretmek we ondan dikeltmek — programmanyň içinden.
15. **Çap edilýän hasabat.** Aýratyn okuwçynyň işi mugallymyň goly üçin ýeri
    bolan resminama görnüşinde çykarylýar.
16. **Okuwçynyň şahsy otagy.** Her okuwçynyň öz hasaby we ähli işleriniň
    taryhy: bal, göterim, ortaça netije. Okuwçy öz ösüşini özi görýär —
    bahalandyrma diňe mugallym üçin däl, okuwçy üçin hem görnükli bolýar.
17. **Mugallymyň otagy.** Taýýarlan materiallary, geçiren sapaklarynyň taryhy
    we öz açar sözüni çalyşmak.
18. **Her mugallym öz materialyna eýe.** Sapaklar, soraglar we synaglar
    mugallyma degişli: başganyňky görünmeýär. Platforma bilen gelýän
    nusgalyk mazmun ählisine açyk, ýöne diňe nusga alyp üýtgedip bolýar.
19. **Ulgam dolandyryjysy ähli hasaplary dolandyrýar.** Ähli mugallymlar we
    ähli okuwçylar bir sanawda: kim haýsy topara we mugallyma degişli,
    kimiň hasaby bar, soňky girişi haçan. Login we açar söz bir basyşda
    berilýär we çap edilýär.
20. **Öý işi.** Synag diňe janly sapakda däl: tabşyryk bellenen möhlete çenli açyk
    durup bilýär. Okuwçy özüne amatly wagtda girýär, möhleti işe başlan
    pursatyndan hasaplanýar. Bu klasdan daşary özbaşdak okamaga mümkinçilik berýär.
6. **Dogruçyllyk.** Möhleti serwer hasaplaýar; dogry jogaplaryň açary synag
   dowam edýärkä okuwçynyň enjamyna ugradylmaýar.
7. **Ýalňyşlyklary seljermek.** Platforma topar boýunça iň köp ýalňyşylan soraglary
   we gaýtadan düşündirilmeli mowzuklary görkezýär.
8. **Bäş ders boýunça interaktiw görkezmeler.** Okuwçynyň hereketi netijä täsir edýär.

## 6. Taslamanyň bilim taýdan ähmiýeti

**Interaktiw görkezme düşündirişi çalyşýar.** Okuwçy parametri üýtgedip, netijäni
öz gözi bilen görýär: parabola nähili üýtgeýär, garşylyk artanda lampa näme üçin
sönýär, reaksiýa haçan deňlenýär. Bu formulany ýat tutmakdan has çuň öwrenmekdir.

**Bahalandyrma okuwa öwrülýär.** Mugallym diňe «3» ýa-da «5» däl-de, synpyň haýsy
mowzukda ýykylandygyny görýär. Şol mowzuk ertirki sapakda gaýtadan düşündirilýär.
Bu — bahalandyrmanyň okuwa hyzmat etmegi.

**Tejribe elýeterli bolýar.** Reaktiwi we enjamy bolmadyk mekdepde okuwçy himiki
reaksiýany we elektrik zynjyryny ekranda synap görýär.

**Deňlik.** Platforma internede bagly bolmansoň, şäherdäki we obadaky synp
deň şertde işleýär.

## 7. Geljekde ösdürmegiň mümkinçilikleri

- Her ders boýunça sapaklaryň we interaktiw modelleriň sanyny artdyrmak.
- Mugallymlaryň öz sapaklaryny alyşmagy üçin ýerli kitaphana.
- Žurnaly mowzuk derejesine çenli çuňlaşdyrmak: haýsy mowzukda öňegidişlik bar.
- Aýratyn bilim zerurlyklary bolan okuwçylar üçin ses bilen okamak we ulaldylan görnüş.
- Žurnaly PDF görnüşinde eksport etmek.
- Mekdebiň serwerinde birnäçe synpyň bilelikde işlemegi.
- Ýokary okuw mekdepleri üçin uýgunlaşdyrmak: leksiýa, kollokwium, özbaşdak iş.

---

## Kabul ediş şertleri boýunça ýagdaý

Aşakdaky sanlar **ölçelen netijelerdir** — 30 okuwçynyň bir wagtda işlemegi
synag edildi (600 jogap).

| Görkeziji | Talap | Ölçelen netije |
|---|---|---|
| Ýerli router arkaly işlemek | 1 mugallym + 30 okuwçy | ✓ 30 okuwçy bir wagtda, 600 jogap 0,2 sekuntda |
| Sapaga goşulmak | 1 minutda | ✓ 30 okuwçy 0,07 sekuntda goşuldy |
| Jogaby saklamak | ≤2 sekunt | ✓ ortaça **8 ms**, 95% ýagdaýda **14 ms** |
| Netijäni çykarmak | 30 okuwçy × 20 sorag, ≤5 sekunt | ✓ **2 ms** |
| Ekran paýlaşma | 720p, ≥5 kadr/sek, ≤2 sek gijikme | JPEG kadrlary WebSocket arkaly, 5 kadr/sek |
| Hotspot arkaly işlemek | 5 hakyky enjamda iş akymy | Brauzer arkaly, goşmaça programma gerek däl |
| Birikmäni dikeltmek | 60 sekuntdan soň ýitgisiz | Ýerli nobat + `client_key` arkaly gaýtalanmazlyk |

> Talap edilýän çäkler bilen ölçelen netijäniň arasynda uly ätiýaç bar:
> jogaby saklamak talapdan **140 esse**, netijäni çykarmak **2500 esse** çalt.

---

## Görkezme meýilnamasy (5–7 minut)

1. **Internedi öçürmek.** Platforma işlemegini dowam etdirýär.
2. **Sapagy başlatmak.** Ekranda 5 harply kod peýda bolýar.
3. **Eminleri çagyrmak.** Olar telefonlary bilen hotspot-a birigip, koda goşulýarlar.
4. **Interaktiw görkezme.** Fizika: garşylygy üýtgedip, lampanyň sönmegini görkezmek.
5. **Ekrany görkezmek.** Mugallymyň ekrany olaryň telefonlarynda peýda bolýar.
6. **Synag.** Eminler birnäçe soraga jogap berýärler; panelde jogaplar janly görünýär.
7. **Tory kesmek.** Telefonda Wi-Fi öçürilýär, jogap ýazylýar — «iberilmegine
   garaşylýar». Wi-Fi gaýtadan açylýar — jogap awtomatik ugradylýar.
8. **Netijeler.** Tablisa, deslapky bellik we «gaýtadan düşündirmeli mowzuklar».
