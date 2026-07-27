run:
	go run ./cmd/main.go

mseed:
	go run ./seed/

nuke:
	rm -rf database.db

# Las pruebas usan su propia base temporal, asi que no tocan database.db ni la
# instancia remota: no hace falta configurar nada para ejecutarlas.
test:
	go test ./...

# Igual que test, pero mostrando cada caso. Util para ver que se esta cubriendo.
test-v:
	go test ./... -v

# Cada prueba declara con que tecnica se diseño (caja negra o caja blanca) con
# testutil.Marcar. Estos tres objetivos leen esas marcas de la salida:
#
#   test-tecnicas  las lista todas, agrupadas por paquete, con el balance final
#   test-negra     solo las de caja negra: derivadas del contrato
#   test-blanca    solo las de caja blanca: derivadas de la estructura del codigo
#
# La marca aparece solo con -v, que es la convencion de Go: `make test` es silencioso.
test-tecnicas:
	@go test ./... -v 2>&1 | grep -E '▪ CAJA|── técnicas|^   CAJA|^                 ·|^(ok|FAIL|---)' || true

test-negra:
	@go test ./... -v 2>&1 | grep '▪ CAJA NEGRA' | sed 's/^ *//' || true
	@go test ./... -v 2>&1 | grep -c '▪ CAJA NEGRA' | xargs -I{} echo "→ {} funciones de caja negra"

test-blanca:
	@go test ./... -v 2>&1 | grep '▪ CAJA BLANCA' | sed 's/^ *//' || true
	@go test ./... -v 2>&1 | grep -c '▪ CAJA BLANCA' | xargs -I{} echo "→ {} funciones de caja blanca"

# Cobertura de todo el codigo de la aplicacion, no solo del paquete en prueba:
# -coverpkg cuenta lo que las pruebas de la API ejercitan en las capas de abajo.
cover:
	go test ./... -coverpkg=./... -coverprofile=cover.out
	go tool cover -func=cover.out | tail -1

# Abre el detalle de cobertura por linea en el navegador.
cover-html: cover
	go tool cover -html=cover.out

check:
	go build ./...
	go vet ./...
	go test ./...

.PHONY: run mseed nuke test test-v test-tecnicas test-negra test-blanca cover cover-html check
