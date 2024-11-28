# HKA Go-Fuzzing Seminar (WS 24/25)

Diese Seminararbeit ist Benstandteil des Go-Fuzzing Seminars (WS 24/25).

Fuzz-Tests sind seit Version 1.19 ein Bestandteil der Programmiersprache go.
Go ermöglicht somit das Testen von Funktionen mit "zufälligem" Input.
Input wird zur Laufzeit des Tests generiert, dabei versucht die Fuzz-Engine alle möglichen
Ablaufpfade im Code zu finden und auszuführen (guided-fuzzing).

Es wird jedoch nicht das Erzeugen von beliebigem Input ermöglicht.
Nur primitive Datentypen (`bool`, `byte`, `float32`, `float64`, `int`, `int8`, ...), sowie
die Typen `string` und `[]byte` werden unterstützt.

Im ursprünglichen Desgin-Draft war auch eine Unterstützung von Structs und Typen, welche
`TextMarshaler` und `TextUnmarshaler` bzw. `BinaryMarshaler` und `BinaryUnmarshaler` vorgsehen.
Dies würde zu einer Reduktion von trivialem Code, wie das Erzeugen/Initialisieren von Structs etc.
vereinfachen.

## Existierende Lösungen

Viele bereits existierenden Fuzz-Test Lösungen für go unterstützen das Testen mit diesen Typen.
Für [go-fuzz](https://github.com/dvyukov/go-fuzz), einen beliebten go-Fuzzer, gibt es beispielweise
[go-fuzz-utils](https://github.com/trailofbits/go-fuzz-utils), eine Erweiterung die Fuzztesting sowowhl
mit Structs, aber auch Arrays und Maps ermöglicht.

Ein anderes Beispiel ist die Bibliothek [go-fuzz-headers](https://github.com/AdaLogics/go-fuzz-headers/tree/main).
Diese bietet, im Rahmen der nativen Fuzz-Tests, Unterstützung für Structs, Arrays und Maps.
Der folgende Beispielcode ist, in leicht geänderter Form,
diesem [Blog](https://adalogics.com/blog/structure-aware-go-fuzzing-complex-types)
entnommen:

```go
package fuzzing

import (
	"testing"
	fuzz "github.com/AdaLogics/go-fuzz-headers"
)

func Fuzz(f *testing.F) {
	//this test tests nothing
	//a new Struct is created a populated with random values
	//noting else happens
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzConsumer := fuzz.NewConsumer(data)
		targetStruct := &Demostruct{}
		err := fuzzConsumer.GenerateStruct(targetStruct)
		if err != nil {
			return
		}
		//TODO do things with targetStruct
		// targetStruct.doThings()...
	})
}
```

## Probleme der Lösungen

Neben den bereits gennaten Möglichkeiten zum Erzeugen zufälliger Test-Daten gibt es auch viele weitere Bibliotheken.
Alle dieser Lösungen haben allderings eine Sache gemeinsam, nämlich das generieren der Daten basierend auf einem Wert
vom Typ `[]byte`. Einige der Bytes werden zum initialisieren eines pseudo-Zufallsgenerators verwendet, der Rest wird zum
Erzuegen den benötigten Daten verwendet. Sind alle Werte erschöpft, können über den Zufallsgenerator weitere Bytes
erzeugt werden.

Dieser Ansatz lässt sich nicht optimal mit guided-fuzzing kombinieren, da der Zusammenhang zwischen dem ursprünglichen
Parameter (Byte-Array) und den erzeugten Daten, nicht trivial ist, und die Konsequenzen von Änderungen am Input durch
den pseudo-Zufall schwer zu berechnen sind. Zudem wird auch der Code, der die Daten generiert, als Bestandteil des
Tests betrachtet, und es wird versucht alle möglichen Ausführuingspfade in diesem Code zu finden.

## Das Problem nativer Tests

Der folgende Code-Ausschnitt zeigt das Grundgerüst eines nativen Fuzz-Tests in go.

```go
package fuzzing

import (
	"testing"
)

// ein neues Struct mit 2 Feldern
type Person struct {
	Name  string
	Alter int
}

//alle Fuzz-Tests beginnen mit dem Prefix-Fuzz und erwarten als einzigen Parameter
//ein Object vom Typ *testing.F
func FuzzSomeTest(f *testing.F) {

	//mit f.Add wird der sogenante Test-Corpus befüllt, die Werte im Corpus sind der
	//Ausgangspunkt für das Erzeugen neuer Werte für den Test
	f.Add(23, "Maxi")

	// der Testkorpus kann mit mehr als einem Wert befüllt werden
	f.Add(23, "Jemand")

	//der eigentliche "Test" wird hier definiert
	//f.Fuzz erwartet eine Funktion mit der selben Typ-Signatur, wie die Werte im Test-Corpus
	f.Fuzz(func(t *testing.T, alter int, name string) {

		//erzeuge ein Struct aus den gegebenen Parametern
		person := Person{name, alter}

		//einTest erwartet ein Struct vom Typ Person. 
		//ist die Person ungültig, wird ein Fehler geworfen
		einTest(person)
	})
}
```

Man kann sich leicht vorstellen, dass der Code zum initialisieren eines mehrfach verschatelten Structs nicht komplex
aber verbos ist. Ähnliches gilt für das Initalisieren größerer Arrays oder Maps.

## Lösungsvorschlag: Zerlegen und wieder zusammenbauen

Es soll ermöglicht werden, Structs, Array und Maps zum Test-Corpus hinzuzufügen. Zudem sollen auch neue, zufällige
Instanzen dieser Objete erzeugt werden können. Dazu wird der folgende Lösungsansatz verwendet:

Die Objekte werden in ihre einzelnen Werte zerlegt, diese Werte werden dann, für den Nutzer verseteckt an `f.Add(...)`
weitergegeben. Ein `Person` Struct , aus dem vorherigen Abschnitt, wird dann als ein `string` und ein `int` Feld
behandelt.<br/>
Die später im Test zufällig erzeugten Argumente werden dann, wieder in ein `Person` Struct transformiert.
Ein Array der Länge 3 wird dann in 3 Felder zerlget und die Werte werden zum Test-Corpus hinzugefügt.
Dementsprechend werden bei der Testausführung 3 neue Werte erzeugt und wieder in einem Array vereingigt. Für Maps wird
das selbe Prinziep verwendet.

Dieser Ansatz hat allerdings auch Nachteile:

* funktioniert nur mit Structs bei denen alle Felder öffentlilch sind
* zyklische Structs können nicht getestet werden (andere Fuzzer sollten auch keine zyklischen Structs erzeugen)
* es werden nur zufällige Arrays und Maps einer Länge erzeugt
* Reflection ist "langsam"

# Implementierung

## Unterstützung für Structs

Wie bereits im vorherigen Abschnitt beschrieben, ermöglicht dieser Ansatz nur das Testen von Structs, bei denen
alle Felder öffentlich sind. Es folgen Beispiele für unterstütze und nicht unterstütze Arten von Structs.

 ```go
 // alle Felder sind öffentlich 
 // => funktioniert 
type goodStruct struct {
    First int
    Second string
}

// alle Felder sind öffentlich, 
// geschachtelte Structs haben auch nur öffentliche Felder
// => funktioniert
type goodNestedStruct struct {
    Nested goodStruct
    Val int
}

// ein Feld ist nicht öffentlich
// funktioniert nicht
type badStruct struct {
    first int
    Second string
}
 ```

Die Syntax für die Funktionserweiterung der nativen Fuzz-Tests soll sich nach Möglichkeit kaum von der Syntax nativer
Fuzz-Tests abweichen. Deswegen wird die Erweitung über ein Struct `FuzzPlus` implementiert, welches einen Wert vom Typ
`*testing.F` umschließt. Dadurch sind alle nicht explizit für `FuzzPlus` überschriebenen Methoden von `*testing.F` auch
für `FuzzPlus` verfügbar.

Die Methoden `Add(...)` und `Fuzz(...)` werden überschrieben, damit übergebene Werte entsprechend transformiert werden
können. Beispielsweise der bereits aus einem vorherigen Abschnitt bekannte `FuzzSomeTest` kann somit wie folgt
vereinfacht werden:

```go
func FuzzSomeTest(f *testing.F) {

    //hier wird ein neues FuzzPlus Struct erzeugt
    ff := NewFuzzPlus(f)
    
    //FuzzPlus.Add(...) funktioniert nach außen hin exakt so wie testing.F.Add(...)
    ff.Add(Person{"Maxi", 23})
    ff.Add(Person{"Jemand", 23})
    
    //FuzzPlus.Fuzz(...) funktioniert nach außen hin exakt so wie testing.F.Fuzz(...)
    ff.Fuzz(func (t *testing.T, person Person) {
        einTest(person)
    })
}
```

Das Zusammenbauen der Structs wird an FuzzPlus ausgelagert. Dadurch wird der Test übersichtlicher und kürzer.

### Funktionsweise

Wie bereits bekannt benötigt jeder Fuzz-Test einen Test-Corpus, welcher über `Add(...)` mit primitiven Datentypen,
strings und Byte-Arrays befüllt werden kann. An `FuzzPlus` übergebene Structs werden, per Reflection, in ihre
Komponenten zerlegt. Diese Komponetnen werden entweder direkt zu `Add(...)` hinzugefügt, oder, wenn es sich um Structs
handelt, weiter zerlegt (deswegen können zyklische Structs nicht zum Test-Corpus hinzugefügt werden). Verschachtelte
Structs werden wie ein Baum in-order traversiert.

Bei einem normalen Fuzz-Test wird die `Fuzz(...)` übergebene Funktion f direkt mit den generierten Werten aufgerufen.
Das ist jetzt nicht mehr möglich, da die Argumente nur als Liste von einzelnen Werten (und nicht als Structs) vorliegen.
Die Argumente der Funktion f werden, per reflection, ermittelt. Mit den Typen ist es jetzt möglich, die List an Werten
in eine Liste von zusammengsetzten Typen zu überführen.

Hier ein Beispiel:

```
origin  := [int, MyStruct{string, int}, OtherStruct{bool, NestedStruct{string}, bool}, int8] <= original
//wird durch Auflösen der Structs zu:
flattend :=[int,          string, int,              bool,              string,  bool,  int8] <= aufgelöst
//              { MyStruct           } {OtherStruct       {NestedStruct      }      }        <= Herkunft der Werte
//flattend is added to corups: 
testing.F.Add(flattend)
//Aus func(testing.T, int, MyStruct{string, int}, OtherStruct{bool, NestedStruct{string}, bool}, int8) wird generiert:
//NOTE: param names are omitted
testing.F.Fuzz(func(t testing.T, int, string, int, bool, string, bool, int8)) 
```

### Performance Vergleich native Fuzz-Tests, FuzzPlus und fuzz-headers Bibliothek

Im folgenden wird die Performance von `FuzzPlus` und der fuzz-headers Bibliothekt verglichen. Zudem sind auch die Werte
für native Fuzz-Tests aufgeführt, damit die Ergebnisse besser eingeordnet werden können.
Als relevante Größe für die Performance wird die Zeit, bis ein Fehler entdeckt wird angenommen. Andere Größen, wie z.B.
die Anzahl der benötigten Testdurchläufe, hängen stark mit der benötigten Zeit zusammen, sind aber keine
ausschlaggebenden Größen.

Der Test wird wie folgt aufgesetzt: <br/>
Es gibt zwei Funktionen `F1(x) = x^3+4*x^2-2` und `F2(x) = x^4-1`. Der Test soll fehlschlagen, wenn `x1` und `x2` so
gewählt werden, dass `F(x1) - F(x2) < 0.001`.

Nach 20 Testdurchläufen steht fest: <br/>
Die nativen Tests benötigen im Durchschnitt die wenigsten Versuche um einen Fehler zu finden und sind mit
durchschnittlich 4.25s am schnellsten. Mit `FuzzPlus` durchgeführte Tests benötigten im Durchschnitt mit 7.05 ca 60%
länger. Die Tests, die mit der fuzz-headers Bibliothek durchgeführt wurden benötioten durchschnittlich 131.35s und somit
mehr als 30 mal mehr Zeit. Dabei waren auch wesentlich mehr Versuche notwendig um einen Fehler zu finden. <br/>
Auch interessant ist die Tatsache, dass sowohl die nativen, als auch die `FuzzPlus` Tests unterschiedliche Fehler
erzeugende Wertepaare finden konnten, die fuzz-headers-Bibliothek jedoch stets das selbe Wertepaar entdeckt hat.

Die Untschiede in der Ausführungszeit und auch bei der Anzahl der Testdurchläufe entsprechen den Erwartungen.
Da die nativen Tests keinen zusätzlichen Code ausführen müssen. Die `FuzzPlus` tests sind langsamer, da bei jedem
Durchlauf Stucts hergestellt werden müssen, was das Ausführen von reflection-lastigem Code bedingt. Die Anzahl der
benötigten Testdurchläufe ist ähnlich zu den nativen Tests, da der guided Fuzzer nicht viele weitere Wege zu
berücksichtigen hat. Die fuzz-headers Bibliothek performt am schlchtetsten, da teilweise ungültige Bytefolgen erzeugt
werden, was einen Abbruch des Versuchs erzwingt, weil das Erzeugen der Zufallswerte Komplexer ist und nicht die native
Unterstützung dafür genutzt werden kann, weil die Verwendung von Pseudozufall es dem Fuzzer erschwert
unterschiedliche Ausführungspfade zu unterusuchen und weil der Code zum Erzeugen der Werte Bestandteil des Tests ist,
was beduetet, dass der Fuzzer auch in diesem Code nach möglichen Ausführungspfaden sucht.

```go
func FuzzNative(f *testing.F) {
    var counter int64 = 0
    
    f.Add(float64(0), float64(0))
    f.Add(float64(0), float64(1))
    f.Add(float64(-1), float64(0))
    
    f.Fuzz(func (t *testing.T, x1 float64, x2 float64) {
        runNumber := atomic.AddInt64(&counter, 1)
        if Similar(Holder{x1, x2}) {
            t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, x1, x2)
        }
    })
}

func FuzzMyStruct(f *testing.F) {
    ff := NewFuzzPlus(f)
    
    var counter int64 = 0
    
    ff.Add(Holder{0, 0})
    ff.Add(Holder{0, 1})
    ff.Add(Holder{-1, 0})
    
    ff.Fuzz(func (t *testing.T, h Holder) {
        runNumber := atomic.AddInt64(&counter, 1)
        if Similar(h) {
            t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, h.X1, h.X2)
        }
    })
}

func FuzzFuzzHeaders(f *testing.F) {

    var counter int64 = 0
    
    f.Fuzz(func (t *testing.T, data []byte) {
        
        fuzzConsumer := fuzz.New    Consumer(data)
        h := &Holder{}
        err := fuzzConsumer.GenerateStruct(h)
        if err != nil {
            //return if an error constructing the struct happens
            return
        }
        runNumber := atomic.AddInt64(&counter, 1)
        
        if Similar(*h) {
            t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, h.X1, h.X2)
        }
    })
}
```

## Unterstützung für Arrays

Die Unterstüzung von Arrays erfordert das Speichern zusätzlicher Daten (im folgenden Metainformationen). Das liegt
daran, das von einem Typ `[]T` nicht auf die erforderliche Länge geschlossen werden kann. Zudem müssen zusätzliche Fälle
wie mehrdimensionale Arrays, Arrays von Structs, Structs mit Arrays als Felder und Kombinationen dieser Fälle beachtet
werden.

Diese Informationen müssen beim Hinzufügen der Werte zum Test-Korupus, bzw. beim Aufruf der Funktion `FuzzPlus.Add(...)`
ermittelt werden. Jedes, zum Test-Korpus hinzugefügte, n-Tupel muss die selben Metainformationen erzeugen, da sonst das
Zusammensetzen der Strucs und Arrays fehlschlägt. 

### Funktionsweise - Idee

`FuzzPlus` wird um ein Feld zum Speichern von Metainformationen erweitert. Die Metainformationen geben an, an welcher
Stelle im Werte-Vektor ein Array beginnt und endet und aus wie vielen Elementen des Werte-Vektors die einzelnen Elemente
des Arrays bestehen.

```go
type FuzzPlus struct {
	*testing.F
	arrays []ArrayPosition
}

type ArrayPosition struct {
    Start      int
    End        int
    TypeLength int
}
```

Zudem wird auf `FuzzPlus` eine neue Funktion zum Hinzufügen von Werten zum Test-Korpus, mit Generierung von 
Metainforamtionen, `Add2(...)` definiert. Dadurch ist es möglich, mehrere n-Tupel zum Korpus hinzuzufügen, wobei nur 
einmal die Metainformationen berechnet werden müssen.

```go
ff.Add2([][]int{{1, 2}, {3, 4}}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
//this would Produce the following Metainformation
{ArrayPosition{0, 3, 1}, ArrayPosition{0, 1, 1}, ArrayPosition{2, 3, 1}, ArrayPosition{4, 3, 1}, ArrayPosition{4, 3, 1}, ArrayPosition{5, 6, 1}}
```

### Probleme bei der Implementierung

Leider funktionieren nicht alle der oben genannten Fälle. Auch die Fälle, die hier als funktionierend aufgelistet 
werden, könnten bei anderen Eingabedaten fehlschlagen.

Zudem ist es fragwürdig, ob das Erzeugen von Arrays mit fester Länge tatsächlich ein sinnvoller Use-Case ist. Wenn eine
Implementierung für Arrays unterschiedlicher Länge getestet werden soll dann ist ein solcher Fuzz-Test nicht hilfreich.
Die Länge von Arrays kann ein wichtiger Faktor sein, welcher mit Fuzz-Tests dieser Art nicht berücksichtigt wird.
Eine Erweiterung des momentanen nativen Fuzzers auf die hier genutze Methode, mit variabler Array Länge ist nicht 
möglich.

#### Fuzzing with 2-dimensionalen arrays - funktioniert

```go
func FuzzPlusPlusEven2(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2([][]int{{1, 2}, {3, 4}}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
    ff.Add([][]int{{-1, -1}, {4, 4}}, []string{}, []string{}, false, []string{"banan", "bonono"}, -14, myStruct{12, "Test"})
    
    ff.Fuzz(func (t *testing.T, in [][]int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {
    
        if in[0][0] == in[1][1] {
    f       mt.Println(in, s, ss, b, strs, i, myStruct2)
            t.Errorf("An Error, how sad")
        }
    })
}
```

#### Fuzzing mit Structs mit Arrays - funktioniert

```go
type ArrayStruct struct {
    Arr []int
    Str string
}

func FuzzPlusPlusEven22(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2(ArrayStruct{[]int{1, 2, 3}, "Hallo"})
    
    ff.Fuzz(func (t *testing.T, arrayStruct ArrayStruct) {
        
        if arrayStruct.Arr[2] == len(arrayStruct.Str) {
            t.Errorf("An Error, how sad")
        }
    })
}

```

#### Fuzzing mit einem Array an Structs - funktioniert

```go
func FuzzPlusPlusEven222(f *testing.F) {

    ff := NewFuzzPlus(f)
    
        ff.Add2([]myStruct{{1, "One"}, {2, "Two"}})
    
    ff.Fuzz(func (t *testing.T, arrayStructs []myStruct) {
    
        if arrayStructs[0].First == len(arrayStructs[1].Second) {
            t.Errorf("An Error, how sad")
        }
    })
}
```

#### Fuzzing mit 2-dimensionalem Array von Structs mit Arrays - funktioniert nicht

```go
func FuzzPlusPlusEven22222(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2([][]ArrayStruct{{{[]int{1}, "a"}, {[]int{2}, "bac"}}, {{[]int{1}, "a"}, {[]int{2}, "bac"}}})
    
    ff.Fuzz(func (t *testing.T, arrayStructs [][]ArrayStruct) {
    
        if arrayStructs[0][0].Arr[0] == len(arrayStructs[0][1].Str) {
            t.Errorf("An Error, how sad")
        }
    })
}
```

## Unterstützung für Maps

Eine mögliche Erweitung des nativen Fuzzers für die Unterstützung von Maps wäre der Erweitung für Arrays sehr ähnlich.
Dementsprechend gelten die Limitierungen, wie eine nicht-variable Länge und die Notwendigkeit der gleichen Länge bei 
geschachtelten Strukturen auch bei Maps.

Da die Implementierung für Arrays momentan nicht vollständig ist, wird keine Unterstützung für Maps umgesetzt.

# Fazit

Es ist möglich den nativen go Fuzzer zu erweitern um das Fuzzen mit neuen Datentypen zu ermöglichen. Eine Limitierung
ist, dass man die zu unterstützenden Typen bijektiv auf eine feste Anzahl an momentan unterstützen Typen abbilden muss.
Eine Erweiterung des in dieser Arbeit entwickelten `FuzzPlus` könnte die Unterstützung von Typen sein, die 
`TextMarshaler`/`BinaryMarshaler` und `TextUnmarshaler`/`BinaryUnmarshaler` implementieren. Dabei ist alledings fraglich
, wie gut der go Fuzzer `string`/`[]byte` Daten generiert, welche sich sinnvoll zu einem Strcut unmarshalen lassen.
Zudem ist zu beachten, dass jede auf Reflection basierende Erweiterung zu einer erheblichen Verschlechterung des
Durchsatzes führt.
