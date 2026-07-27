package testutil

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Tecnica identifica con que enfoque se diseño una prueba.
//
// La distincion no es academica: dice que rompe la prueba cuando falla. Una de
// caja negra falla si cambia el comportamiento que el cliente observa —un codigo
// de respuesta, la forma de la respuesta—; una de caja blanca falla si cambia la
// estructura interna, aunque el comportamiento visible siga igual.
type Tecnica string

const (
	// CajaNegra: el caso se derivo del contrato, sin mirar la implementacion. Solo
	// usa la interfaz publica y comprueba lo que un cliente podria comprobar.
	CajaNegra Tecnica = "CAJA NEGRA"

	// CajaBlanca: el caso se derivo de leer el codigo. Ataca una rama concreta, un
	// mecanismo interno (una transaccion, el borrado logico) o inspecciona el
	// estado guardado, que no es observable desde la interfaz.
	CajaBlanca Tecnica = "CAJA BLANCA"
)

var (
	recuento   = map[Tecnica][]string{}
	recuentoMu sync.Mutex
)

// Marcar declara con que tecnica se diseño la prueba y por que. La marca aparece
// en la salida con `go test -v` (o `make test-v`), asi que la propia ejecucion
// documenta la estrategia:
//
//	▪ CAJA BLANCA · TestCrearAtleta — ataca la transaccion del alta y las claves foraneas
//
// Cuando una prueba mezcla las dos —lo habitual—, se marca la predominante y el
// motivo aclara la parte que corresponde a la otra.
func Marcar(t *testing.T, tecnica Tecnica, motivo string) {
	t.Helper()

	recuentoMu.Lock()
	recuento[tecnica] = append(recuento[tecnica], t.Name())
	recuentoMu.Unlock()

	t.Logf("▪ %s · %s — %s", tecnica, t.Name(), motivo)
}

// ResumenTecnicas devuelve el reparto de las pruebas marcadas del paquete. Se
// imprime desde TestMain al terminar, para cerrar la ejecucion con el balance
// entre los dos enfoques.
func ResumenTecnicas() string {
	recuentoMu.Lock()
	defer recuentoMu.Unlock()

	if len(recuento) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n── técnicas de prueba de este paquete ──\n")

	tecnicas := make([]Tecnica, 0, len(recuento))
	total := 0
	for tecnica, nombres := range recuento {
		tecnicas = append(tecnicas, tecnica)
		total += len(nombres)
	}
	sort.Slice(tecnicas, func(i, j int) bool { return tecnicas[i] < tecnicas[j] })

	for _, tecnica := range tecnicas {
		nombres := recuento[tecnica]
		fmt.Fprintf(&b, "   %-11s %2d de %d funciones (%d%%)\n",
			tecnica, len(nombres), total, 100*len(nombres)/total)
		for _, nombre := range nombres {
			fmt.Fprintf(&b, "                 · %s\n", nombre)
		}
	}
	return b.String()
}
