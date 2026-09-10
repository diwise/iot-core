# Observationskontrakt – flera objekt från samma enhet i ett SenML-pack

Status: Fas A (kontrakt + fixturer). Implementation sker i faserna B–E enligt
`TODO.md` i samlingsmappen. Ändringar här kräver samma granskning som
payload-/routingändringar: producent och samtliga konsumenter måste verifieras
mot fixturerna i `testdata/` innan aktivering.

## 1. Rapport

En rapport är observationerna från **ett inkommande enhetsrapporttillfälle**.
Observationer från separata rapporttillfällen batchas aldrig ihop i efterhand,
inte ens vid retry – retry återanvänder rapportens ursprungliga pack.

- Ett pack innehåller data från exakt **en enhet** (`diwise.Parse` avvisar
  flera enheter).
- En enhet tillhör exakt **en tenant**. Tenant hämtas från betrodd
  enhetsmetadata i core (device-lookup) och får aldrig styras av inkommande
  payload.
- En enhet kan ha flera sensorer och flera sensorer av samma typ.

## 2. Packstruktur

Packet är en sekvens av objektobservationer i trådordning. Varje observation
inleds med en header och följs av sina resurser:

- Header: `<device>[/<kanal>...]/<typ>/0` med `vs` = `urn:oma:lwm2m:ext:<typ>`.
  Exempel: `dev1/3303/0`, `dev1/0/3303/0` (kanal `0`), `dev1/3304/0`.
- Resurser: numeriska namn (`5700`, …) eller metadata under **samma
  fullständiga prefix** som headern. Exempel: `dev1/3303/5700`.
- Ordning och upprepade observationer bevaras. Samma objekt/resurs får
  förekomma flera gånger (t.ex. flera mättidpunkter).
- Relativa tider (`bt`/`t` under 2**28) löses mot en explicit referenstid
  (mottagningstid). Varje observations mättidpunkt bevaras, inklusive
  delsekunder och olika tidpunkter per observation.

Kanalidentitet: pathsegment mellan device och typ, enligt befintlig
konvention (`deviceID + "/0"`, `"/1"`, … för flera sensorer av samma typ).
Enhetsidentitet är alltid **första segmentet** och är densamma för alla
observationer i packet.

## 3. Metadata

- Packnivå: `<device>/tenant`, `<device>/source`, `<device>/env`,
  `<device>/lat`, `<device>/lon`. Packmetadata ärvs av alla objektobservationer.
- Objektnivå: `<prefix>tenant` etc. vid behov; åsidosätter packnivå.
- `tenant` är obligatorisk per observation efter berikning
  (`ValidateAccepted`). Saknad eller motstridig tenant avvisar rapporten.
- Nakna metadatarecords (`tenant`, `source`, `env` utan deviceprefix) får
  **inte** förekomma i accepterade pack – de bryter `diwise.Parse`.
  Mottagare läser metadata via `diwise`-API:t, inte via första-match-uppslag.

Arv och validering följer `senml/diwise`-paketets `Parse`/`overlay`-regler.

## 4. Transport

- Kommando `MessageReceived` → `iot-core` (routingnyckel `iot-core`),
  oförändrat. Cores kommandofilter (prefix `application/vnd.oma.lwm2m`)
  matchar båda content-typerna nedan.
- `message.accepted` → events, things, FIWARE, oförändrat.
- Content-Type:
  - enobjektspack (legacy): `application/vnd.oma.lwm2m.ext.<id>+json`,
    oförändrad.
  - flerobjektspack: `application/vnd.oma.lwm2m+json` (ingen enskild
    objekttyp kan namnges).
- Envelope `{pack, timestamp}` oförändrad; `timestamp` sätts vid
  konstruktion (`time.Now().UTC`) och är transporttid, inte mättid.
- Legacy-enobjektspack accepteras genomgående vid sidan av flerobjektspack.

Singulära accessorer (`ObjectID()`, `ContentType()` per objekttyp) är endast
definierade för enobjektspack. För flerobjektspack gäller den generella
content-typen och objektavgränsade uppslag (`Resolved.Get`/`Find`,
`Object.Get`/`Find`, `Pack.Select`).

## 5. Felpolicy

- Agenten validerar varje objektobservation separat mot decoderformatens
  felindikatorer, sentinelvärden och dokumenterade giltighetsgränser (inga
  generella rimlighetsgränser utan sensorstöd). Felaktiga observationer
  loggas strukturerat (enhets-ID, objekt/kanal, felorsak, korrelation – aldrig
  hela sensorpayloads) och utelämnas. Giltiga observationers värden, enheter
  och mättidpunkter ändras inte; bortfall ersätts aldrig med nollvärden.
- Inga giltiga observationer → inget mätmeddelande skickas; bortfallet ska
  vara observerbart (logg/mätetal).
- Rapportgemensamma fel (okänd enhetsidentitet, flera enheter, motstridig
  tenant, strukturskada) hanteras på rapportnivå och stoppar rapporten.
  Device-status-/larmflödet påverkas inte.
- Core: strukturskada och identitets-/tenantfel är permanenta
  (kan aldrig läka vid retry). Hanteringsfel ackas tills idempotens för
  berörda sidoeffekter är verifierad.

## 6. Idempotens och lagring

- Inga nya rapport- eller meddelande-ID:n införs i steg 2. Transportens
  `MessageID`/`CorrelationID` används för felsökning och deduplicering.
- `iot-events` lagrar en rad per post med `id` = fullt recordnamn; redelivery
  skriver över samma `(time, id)`-rad. Varje rad bär sin egen observations
  URN så att tidsserier kan tas ut per sensortyp.
- `iot-events` MQTT publicerar varje observation separat.

## 7. Storlek

Ingen explicit observationsgräns införs initialt; rapporter har naturlig
sensorstorlek (typiskt < 10 observationer). Storleksfördelning följs upp i
Fas F innan någon gräns sätts – en gräns får aldrig tyst tappa observationer.

## 8. Utrullning

Mottagarstöd (core, events, things, FIWARE) i drift före agentens
sammanslagning. Agentens sammanslagning aktiveras kontrollerbart med
dokumenterad rollback. Flerobjektsmeddelanden i kö vid rollback hanteras
enligt utrullningsplanen (Fas F).

## 9. Fixturer

`testdata/` innehåller kanoniska pack som producent och samtliga konsumenter
ska verifieras mot. `observations_test.go` låser tolkningen.

| Fixtur | Innehåll | Förväntat utfall |
|---|---|---|
| `legacy-single-temp.json` | ett objekt `3303`, `5700` | accepteras; en observation |
| `multi-temp-humidity-light.json` | `3303` + `3304` + `3301` från samma enhet, samma `bt` | accepteras; tre observationer med egna URN |
| `multi-channel-temp.json` | två `3303`-observationer, kanal `0` och `1` | accepteras; separata identiteter |
| `repeated-observation.json` | samma objekt/resurs två gånger, olika `t` | accepteras; båda bevaras |
| `multi-object-different-times.json` | två objekt med olika `bt` | accepteras; varsin mättidpunkt |
| `conflicting-tenant.json` | två olika `<device>/tenant` | avvisas |
| `multi-device.json` | poster från två enheter | avvisas |
| `missing-header.json` | resurs utan föregående header | avvisas |
| `empty-pack.json` | `[]` | avvisas (inga observationer) |
