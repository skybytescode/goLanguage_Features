# Reflection și unsafe — Senior

Cum e reprezentat un `reflect.Value`, ce poate și ce nu poate face reflection (câmpuri neexportate, crearea de tipuri noi), regulile exacte pentru `unsafe.Pointer` și uneltele care le verifică: `go vet` și `-d=checkptr`.

## Cum arată un `reflect.Value`

```go,ignore
type Value struct {
	typ_ *abi.Type     // descriptorul de tip din runtime (același ca în interfețe)
	ptr  unsafe.Pointer // pointer la date, sau datele însele dacă încap într-un pointer
	flag                // Kind, plus biți: adresabil, read-only, indirect, metodă
}
```

`reflect.ValueOf(x)` primește un `any`, deci `x` e mai întâi convertit la interfață (posibil cu o alocare, vezi [Interfețe](../../interfete/senior/)); apoi `Value` copiază tipul și pointerul din interfață. Biții din `flag` explică regulile de la nivelul anterior:

- **flagAddr**: valoarea e adresabilă (a venit printr-un `Elem()` pe un pointer sau printr-un câmp al unei valori adresabile) → `CanSet` poate fi adevărat;
- **flagRO**: valoarea a fost obținută printr-un câmp **neexportat** → poate fi citită, dar `Set` dă panic, iar `Interface()` dă panic.

```go
package main

import (
	"fmt"
	"reflect"
)

type Cont struct {
	Public string
	secret string
}

func main() {
	c := Cont{"vizibil", "ascuns"}
	v := reflect.ValueOf(&c).Elem()

	fmt.Println(v.Field(1).String())       // „ascuns”: citirea e permisă
	fmt.Println(v.Field(1).CanSet())       // false: câmp neexportat
	fmt.Println(v.Field(0).CanSet())       // true
	fmt.Println(v.Field(1).CanInterface()) // false: Interface() ar da panic
}
```

## Ce poate crea reflection la rulare

- valori noi de orice tip: `reflect.New(t)`, `reflect.MakeSlice`, `MakeMap`, `MakeChan`;
- **tipuri compuse noi**: `reflect.SliceOf(t)`, `MapOf(k, v)`, `PointerTo(t)`, `FuncOf(...)`, chiar `StructOf(campuri)`;
- funcții noi: `reflect.MakeFunc(tip, implementare)` creează o funcție reală, apelabilă, a cărei implementare primește argumentele ca `[]reflect.Value`.

Ce **nu** poate: crea tipuri cu **metode** noi (`StructOf` nu poate adăuga metode, deci tipurile create nu pot implementa interfețe în afară de cele promovate prin câmpuri încorporate), sau instanția funcții generice (generics sunt rezolvate la compilare).

## Iteratori în `reflect` (Go 1.23)

`Value.Seq()` și `Value.Seq2()` întorc iteratori pentru orice valoare care poate fi parcursă cu `range`: slices, maps, channels, stringuri, întregi și funcții iterator:

```go
package main

import (
	"fmt"
	"reflect"
)

func main() {
	m := map[string]int{"a": 1}
	for k, v := range reflect.ValueOf(m).Seq2() {
		fmt.Println(k.String(), v.Int())
	}
}
```

## Regulile pentru `unsafe.Pointer`

Documentația pachetului `unsafe` enumeră **șase** tipare valide. Orice altceva e comportament nedefinit, chiar dacă „merge” azi:

1. **Conversia `*T1` → `unsafe.Pointer` → `*T2`**, dacă `T2` nu e mai mare decât `T1` și au layout compatibil (ex. `math.Float64bits`).
2. **`unsafe.Pointer` → `uintptr`** (fără conversie înapoi), doar pentru afișare sau hash-uri de adrese.
3. **`unsafe.Pointer` → `uintptr` → aritmetică → `unsafe.Pointer`**, într-o **singură expresie**:

```go,ignore
// VALID: totul într-o expresie
p := unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + unsafe.Offsetof(s.camp))

// INVALID: între cele două linii, GC-ul poate muta sau elibera obiectul
u := uintptr(unsafe.Pointer(&s))
p := unsafe.Pointer(u + unsafe.Offsetof(s.camp))
```

   Mai simplu și sigur din Go 1.17: `unsafe.Add(unsafe.Pointer(&s), off)`.

4. **`uintptr` ca argument la `syscall.Syscall`**, convertit direct în lista de argumente.
5. **Rezultatul lui `reflect.Value.Pointer`/`UnsafeAddr`** convertit imediat la `unsafe.Pointer`.
6. **Conversia `reflect.SliceHeader`/`StringHeader`** — **depreciată**; înlocuită de `unsafe.Slice`, `unsafe.SliceData`, `unsafe.String`, `unsafe.StringData` (Go 1.17–1.20).

Regula din spatele tuturor: un `uintptr` e un **întreg**. GC-ul nu-l vede ca referință, iar stivele se pot muta oricând (vezi [Funcții](../../functii/senior/)). Între momentul în care o adresă devine `uintptr` și cel în care redevine pointer, nu trebuie să existe niciun punct în care runtime-ul să poată interveni.

## Verificări: `go vet` și `checkptr`

- `go vet` (analizorul `unsafeptr`) semnalează conversiile `uintptr` → `unsafe.Pointer` care nu urmează tiparele de mai sus.
- `-gcflags=all=-d=checkptr` instrumentează la rulare conversiile `unsafe.Pointer`: verifică alinierea, că pointerul rezultat rămâne în același obiect alocat și că aritmetica nu iese din el. E **activat automat** cu `-race` și `-msan`.

Verificările `checkptr` privesc obiectele de pe heap: o conversie sau o aritmetică ce duce în afara obiectului alocat oprește programul cu o eroare fatală `checkptr: ...`. Pointerii spre stivă sau spre memorie din afara heap-ului Go nu pot fi verificați, deci absența erorii nu dovedește corectitudinea.

## Promisiunea de compatibilitate nu acoperă `unsafe`

Codul care depinde de layout-ul intern al tipurilor din runtime (structura unui map, câmpurile unui `reflect.Value`, offset-uri în `runtime.g`) se poate strica la orice versiune nouă de Go. Așa s-a întâmplat la trecerea map-urilor la Swiss tables (Go 1.24) cu bibliotecile care citeau direct structura `hmap`. Restricțiile asupra `//go:linkname` din Go 1.23 au același scop (vezi [Pachete și module](../../pachete-module/senior/)).

## cgo și pointerii

La apeluri C, regulile cgo cer ca memoria Go transmisă către C să nu conțină pointeri Go (C nu e urmărit de GC), iar C să nu păstreze pointerii Go după apel. `GODEBUG=cgocheck=1` (implicit) verifică parțial la rulare; `runtime.Pinner` (Go 1.21) fixează obiecte Go ca să poată fi referite din C pe durate mai lungi.

## Pe scurt

- `reflect.Value` = tip + pointer + flag-uri (adresabil, read-only).
- Câmpurile neexportate pot fi citite, dar nu scrise sau extrase ca interfață.
- Reflection poate crea tipuri compuse și funcții, dar nu tipuri cu metode noi.
- Doar cele șase tipare documentate sunt valide pentru `unsafe.Pointer`; `unsafe.Add` și `unsafe.Slice` le înlocuiesc pe cele fragile.
- `go vet` și `checkptr` (automat cu `-race`) prind conversiile greșite.
