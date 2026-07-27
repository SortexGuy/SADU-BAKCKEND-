// Pruebas de integracion de la API: montan el enrutador con el mismo cableado que
// cmd/main.go y ejercitan los endpoints por HTTP contra una base real.
//
// Cubren lo que las pruebas de servicio no pueden ver: los codigos de respuesta,
// la envoltura de la respuesta, donde se aplica la autenticacion y si los nombres
// de los parametros de filtrado del handler coinciden con los que espera el
// cliente.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/handlers"
	"uneg.edu.ve/servicio-sadu-back/internal/middlewares"
	"uneg.edu.ve/servicio-sadu-back/internal/routes"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

const claveDePrueba = "clave-de-prueba-para-firmar"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	// La clave se fija una sola vez: config.SecretKey() la memoriza en la primera
	// consulta, asi que todas las pruebas del paquete deben usar la misma.
	os.Setenv("SECRET_KEY", claveDePrueba)
	codigo := m.Run()
	fmt.Print(testutil.ResumenTecnicas())
	os.Exit(codigo)
}

// api reune el enrutador y los datos que necesitan las pruebas.
type api struct {
	router *gin.Engine
	db     *gorm.DB
	cat    testutil.Catalogo
	token  string
}

// nuevaAPI monta el enrutador igual que cmd/main.go: mismos grupos, mismo
// middleware y en el mismo orden. Asi las pruebas tambien verifican el cableado.
func nuevaAPI(t *testing.T) api {
	t.Helper()

	db := testutil.NuevaDB(t)
	cat := testutil.SembrarCatalogo(t, db)

	athleteService := services.AthleteService{DB: db}
	universityService := services.UniversityServices{DB: db}
	disciplineService := services.DisciplineServices{DB: db}
	majorService := services.MajorServices{DB: db}
	tourneyService := services.TourneyServices{DB: db}
	teacherService := services.TeacherService{DB: db}
	teamService := services.TeamServices{DB: db}
	eventService := services.EventService{DB: db}
	userService := services.UserService{DB: db}

	// Usuario con el que se obtiene el token de las pruebas.
	os.Setenv("ADMIN_USER", "admin")
	os.Setenv("ADMIN_PASS", "clave-de-prueba")
	os.Unsetenv("ADMIN_RESET_PASSWORD")
	testutil.SinError(t, userService.EnsureAdminUser(), "crear el administrador de prueba")

	r := gin.New()
	r.Use(middlewares.RequestID(), middlewares.Recovery())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	routes.RegisterAthletesRoutes(r.Group("/athletes", middlewares.AuthMiddleware()), handlers.NewAthleteHandler(&athleteService))
	routes.RegisterUniversityRoutes(r.Group("/universities", middlewares.AuthMiddleware()), handlers.NewUniversityHandler(&universityService))
	routes.RegisterDisciplines(r.Group("/disciplines", middlewares.AuthMiddleware()), handlers.NewDisciplineHandler(&disciplineService))
	routes.RegisterMajorsRoutes(r.Group("/majors", middlewares.AuthMiddleware()), handlers.NewMajorHandler(&majorService))
	routes.RegisterTourney(r.Group("/tourneys", middlewares.AuthMiddleware()), handlers.NewTourneyHandler(&tourneyService))
	routes.RegisterTeacherRoutes(r.Group("/teachers", middlewares.AuthMiddleware()), handlers.NewTeacherHandler(&teacherService))
	routes.RegisterTeamRoutes(r.Group("/teams", middlewares.AuthMiddleware()), handlers.NewTeamHandler(&teamService))
	routes.RegisterEventsRouters(r.Group("/events"), handlers.NewEventHandler(&eventService))
	routes.RegisterUserRoutes(r.Group("/users"), handlers.NewUserHandler(&userService))

	a := api{router: r, db: db, cat: cat}
	a.token = a.iniciarSesion(t, "admin", "clave-de-prueba")
	return a
}

// llamar ejecuta una peticion contra el enrutador. Con token vacio no envia la
// cabecera de autorizacion.
func (a api) llamar(t *testing.T, metodo, url, token string, cuerpo any) *httptest.ResponseRecorder {
	t.Helper()

	var lector *bytes.Reader
	if cuerpo != nil {
		datos, err := json.Marshal(cuerpo)
		testutil.SinError(t, err, "serializar el cuerpo de la peticion")
		lector = bytes.NewReader(datos)
	} else {
		lector = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(metodo, url, lector)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	respuesta := httptest.NewRecorder()
	a.router.ServeHTTP(respuesta, req)
	return respuesta
}

func (a api) iniciarSesion(t *testing.T, usuario, contrasena string) string {
	t.Helper()

	respuesta := a.llamar(t, "POST", "/users/login", "", map[string]string{
		"username": usuario, "password": contrasena,
	})
	if respuesta.Code != http.StatusOK {
		t.Fatalf("no se pudo iniciar sesion: HTTP %d — %s", respuesta.Code, respuesta.Body.String())
	}

	var envoltura struct {
		Data    string `json:"data"`
		Message string `json:"message"`
	}
	testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer el token")
	return envoltura.Data
}

// datos extrae el campo data de la envoltura como lista.
func lista(t *testing.T, respuesta *httptest.ResponseRecorder) []map[string]any {
	t.Helper()

	var envoltura struct {
		Data    []map[string]any `json:"data"`
		Message string           `json:"message"`
	}
	testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer la lista de la respuesta")
	return envoltura.Data
}

func atletaNuevo(cat testutil.Catalogo, cedula string) map[string]any {
	return map[string]any{
		"FirstNames": "Ana",
		"LastNames":  "Perez",
		"GovID":      cedula,
		"Gender":     string(schema.GenderF),
		"Enrolled":   true,
		"MajorID":    cat.CarreraID,
	}
}

// ── Contrato de la respuesta ────────────────────────────────────────────────

func TestEnvolturaDeLaRespuesta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "la forma de la respuesta tal como la consume el cliente")

	a := nuevaAPI(t)

	t.Run("el exito trae data y message", func(t *testing.T) {
		respuesta := a.llamar(t, "GET", "/athletes", a.token, nil)
		testutil.Igual(t, respuesta.Code, http.StatusOK, "codigo de respuesta")

		var envoltura map[string]any
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer la envoltura")
		if _, existe := envoltura["data"]; !existe {
			t.Error("la respuesta de exito deberia traer el campo data")
		}
		if _, existe := envoltura["message"]; !existe {
			t.Error("la respuesta de exito deberia traer el campo message")
		}
	})

	t.Run("el error trae el formato de problema", func(t *testing.T) {
		respuesta := a.llamar(t, "GET", "/athletes/999", a.token, nil)

		var problema struct {
			Type     string `json:"type"`
			Title    string `json:"title"`
			Status   int    `json:"status"`
			Detail   string `json:"detail"`
			Instance string `json:"instance"`
		}
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &problema), "leer el problema")
		testutil.Igual(t, problema.Status, respuesta.Code, "el status del cuerpo coincide con el HTTP")
		testutil.Igual(t, problema.Instance, "/athletes/999", "la instancia es la ruta pedida")
		if problema.Title == "" || problema.Detail == "" {
			t.Error("el error deberia traer titulo y detalle")
		}
	})

	t.Run("cada respuesta trae su identificador de peticion", func(t *testing.T) {
		respuesta := a.llamar(t, "GET", "/health", "", nil)
		if respuesta.Header().Get("X-Request-Id") == "" {
			t.Error("la respuesta deberia traer la cabecera X-Request-Id")
		}
	})
}

// ── Autenticacion ──────────────────────────────────────────────────────────

func TestAutenticacionDeLosEndpoints(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "que exige token y que es publico, sin mirar como se implementa")

	a := nuevaAPI(t)

	t.Run("los recursos protegidos exigen token", func(t *testing.T) {
		protegidos := []string{"/athletes", "/teachers", "/teams", "/universities", "/disciplines", "/majors", "/tourneys"}
		for _, ruta := range protegidos {
			respuesta := a.llamar(t, "GET", ruta, "", nil)
			testutil.Igual(t, respuesta.Code, http.StatusUnauthorized, "sin token en "+ruta)
		}
	})

	t.Run("con token responden", func(t *testing.T) {
		respuesta := a.llamar(t, "GET", "/athletes", a.token, nil)
		testutil.Igual(t, respuesta.Code, http.StatusOK, "con token en /athletes")
	})

	t.Run("la consulta de eventos es publica", func(t *testing.T) {
		testutil.Igual(t, a.llamar(t, "GET", "/events", "", nil).Code, http.StatusOK, "GET /events sin token")
	})

	t.Run("la escritura de eventos exige token", func(t *testing.T) {
		respuesta := a.llamar(t, "POST", "/events/create", "", map[string]any{"Name": "X"})
		testutil.Igual(t, respuesta.Code, http.StatusUnauthorized, "POST /events/create sin token")
	})

	t.Run("health es publico", func(t *testing.T) {
		testutil.Igual(t, a.llamar(t, "GET", "/health", "", nil).Code, http.StatusOK, "GET /health")
	})

	t.Run("un token invalido se rechaza", func(t *testing.T) {
		testutil.Igual(t, a.llamar(t, "GET", "/athletes", "token-falso", nil).Code,
			http.StatusUnauthorized, "token invalido")
	})
}

// ── Ciclo completo de un recurso ────────────────────────────────────────────

func TestCicloDeVidaDeUnAtleta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "crear, listar, ver, editar y eliminar usando solo la API")

	a := nuevaAPI(t)

	// Crear
	respuesta := a.llamar(t, "POST", "/athletes/create", a.token, atletaNuevo(a.cat, "60000001"))
	testutil.Igual(t, respuesta.Code, http.StatusOK, "crear atleta")

	var creacion struct {
		Data struct {
			ID uint `json:"ID"`
		} `json:"data"`
	}
	testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &creacion), "leer el atleta creado")
	if creacion.Data.ID == 0 {
		t.Fatal("la respuesta de creacion deberia traer el identificador")
	}
	id := creacion.Data.ID

	// Listar
	testutil.Igual(t, len(lista(t, a.llamar(t, "GET", "/athletes", a.token, nil))), 1, "atletas listados")

	// Detalle
	testutil.Igual(t, a.llamar(t, "GET", "/athletes/"+idTexto(id), a.token, nil).Code,
		http.StatusOK, "detalle del atleta")

	// Editar
	editar := atletaNuevo(a.cat, "60000001")
	editar["FirstNames"] = "Ana Maria"
	testutil.Igual(t, a.llamar(t, "PUT", "/athletes/edit/"+idTexto(id), a.token, editar).Code,
		http.StatusOK, "editar atleta")

	// Eliminar
	testutil.Igual(t, a.llamar(t, "DELETE", "/athletes/delete/"+idTexto(id), a.token, nil).Code,
		http.StatusOK, "eliminar atleta")
	testutil.Igual(t, len(lista(t, a.llamar(t, "GET", "/athletes", a.token, nil))), 0, "atletas tras eliminar")
}

// ── Codigos de error ───────────────────────────────────────────────────────

func TestCodigosDeError(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el codigo HTTP que corresponde a cada situacion invalida")

	a := nuevaAPI(t)
	testutil.SilenciarLogs(t)

	testutil.SinError(t, nil, "")
	primero := a.llamar(t, "POST", "/athletes/create", a.token, atletaNuevo(a.cat, "61000001"))
	testutil.Igual(t, primero.Code, http.StatusOK, "crear el primer atleta")

	casos := []struct {
		nombre   string
		metodo   string
		url      string
		cuerpo   any
		esperado int
	}{
		{
			nombre: "cedula duplicada", metodo: "POST", url: "/athletes/create",
			cuerpo: atletaNuevo(a.cat, "61000001"), esperado: http.StatusConflict,
		},
		{
			nombre: "sin cedula", metodo: "POST", url: "/athletes/create",
			cuerpo: atletaNuevo(a.cat, ""), esperado: http.StatusBadRequest,
		},
		{
			nombre: "sin carrera", metodo: "POST", url: "/athletes/create",
			cuerpo:   map[string]any{"FirstNames": "X", "LastNames": "Y", "GovID": "61000002"},
			esperado: http.StatusBadRequest,
		},
		{
			nombre: "carrera inexistente", metodo: "POST", url: "/athletes/create",
			cuerpo:   map[string]any{"FirstNames": "X", "LastNames": "Y", "GovID": "61000003", "MajorID": 999},
			esperado: http.StatusBadRequest,
		},
		{
			nombre: "atleta inexistente al eliminar", metodo: "DELETE", url: "/athletes/delete/999",
			esperado: http.StatusNotFound,
		},
		{
			nombre: "atleta inexistente al consultar", metodo: "GET", url: "/athletes/999",
			esperado: http.StatusNotFound,
		},
		{
			nombre: "evento inexistente", metodo: "GET", url: "/events/999",
			esperado: http.StatusNotFound,
		},
		{
			nombre: "profesor con cedula duplicada", metodo: "POST", url: "/teachers/create",
			// El catalogo ya trae un profesor con esta cedula.
			cuerpo:   map[string]any{"FirstNames": "Z", "LastNames": "W", "GovID": "20000001"},
			esperado: http.StatusConflict,
		},
		{
			nombre: "profesor sin cedula", metodo: "POST", url: "/teachers/create",
			cuerpo:   map[string]any{"FirstNames": "Z", "LastNames": "W"},
			esperado: http.StatusBadRequest,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := a.llamar(t, caso.metodo, caso.url, a.token, caso.cuerpo)
			if respuesta.Code != caso.esperado {
				t.Errorf("se obtuvo HTTP %d y se esperaba %d — cuerpo: %s",
					respuesta.Code, caso.esperado, strings.TrimSpace(respuesta.Body.String()))
			}
		})
	}
}

// ── Filtros por parametros de consulta ─────────────────────────────────────

func TestFiltrosPorParametrosDeConsulta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "que el nombre de cada parametro sea el que envia el cliente")

	// Estas pruebas comprueban que el nombre del parametro que lee el handler es
	// el que usa el cliente: un cambio en uno de los dos lados dejaria el filtro
	// sin efecto en silencio.
	a := nuevaAPI(t)

	ana := atletaNuevo(a.cat, "62000001")
	ana["FirstNames"], ana["LastNames"] = "Ana", "Perez"
	testutil.Igual(t, a.llamar(t, "POST", "/athletes/create", a.token, ana).Code, http.StatusOK, "crear a Ana")

	beto := atletaNuevo(a.cat, "63333333")
	beto["FirstNames"], beto["LastNames"], beto["Gender"] = "Beto", "Quintero", string(schema.GenderM)
	testutil.Igual(t, a.llamar(t, "POST", "/athletes/create", a.token, beto).Code, http.StatusOK, "crear a Beto")

	casos := []struct {
		url       string
		esperados int
	}{
		{"/athletes", 2},
		{"/athletes?search=Ana", 1},
		{"/athletes?search=Quintero", 1},
		{"/athletes?search=63333333", 1},
		{"/athletes?search=zzz", 0},
		{"/athletes?gender=Femenino", 1},
		{"/athletes?gender=Masculino", 1},
		{"/athletes?search=Ana&gender=Masculino", 0},
		{"/athletes?name=Beto", 1},
		{"/athletes?last_name=Perez", 1},
		{"/athletes?gov_id=62000001", 1},
		{"/teachers?search=Ramirez", 1},
		{"/teachers?search=zzz", 0},
		{"/teams?category=Masculino", 1},
		{"/teams?regular=true", 1},
		{"/teams?name=Alfa", 1},
		{"/tourneys?name=Copa", 1},
		{"/tourneys?status=Activo", 1},
		{"/universities?name=UNEG", 1},
		{"/universities?local=true", 1},
		{"/disciplines?name=Voleibol", 1},
		{"/majors?name=Informatica", 1},
	}

	for _, caso := range casos {
		t.Run(caso.url, func(t *testing.T) {
			respuesta := a.llamar(t, "GET", caso.url, a.token, nil)
			testutil.Igual(t, respuesta.Code, http.StatusOK, "codigo de respuesta")
			testutil.Igual(t, len(lista(t, respuesta)), caso.esperados, "resultados")
		})
	}
}

// ── Eventos ────────────────────────────────────────────────────────────────

func TestEventosPorHTTP(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "crear sin torneo como lo hace el formulario, y el detalle como objeto")

	a := nuevaAPI(t)
	testutil.SilenciarLogs(t)

	eventoBase := func() map[string]any {
		return map[string]any{
			"Name":                 "Alfa vs Beta",
			"Date":                 "2026-08-15T10:00:00Z",
			"Status":               "Pendiente",
			"HomeTeamID":           a.cat.EquipoLocalID,
			"OppositeTeamID":       a.cat.EquipoVisitID,
			"ResponsableTeacherID": a.cat.ProfesorID,
			"DisciplineID":         a.cat.Disciplina1ID,
		}
	}

	t.Run("se puede crear sin torneo, como lo envia el formulario", func(t *testing.T) {
		respuesta := a.llamar(t, "POST", "/events/create", a.token, eventoBase())
		testutil.Igual(t, respuesta.Code, http.StatusOK, "crear evento sin torneo")
	})

	t.Run("con un torneo inexistente responde 400", func(t *testing.T) {
		cuerpo := eventoBase()
		cuerpo["TourneyID"] = 999
		respuesta := a.llamar(t, "POST", "/events/create", a.token, cuerpo)
		testutil.Igual(t, respuesta.Code, http.StatusBadRequest, "torneo inexistente")
	})

	t.Run("el detalle devuelve un objeto, no una lista", func(t *testing.T) {
		testutil.Igual(t, a.llamar(t, "POST", "/events/create", a.token, eventoBase()).Code,
			http.StatusOK, "crear evento")

		respuesta := a.llamar(t, "GET", "/events/1", "", nil)
		testutil.Igual(t, respuesta.Code, http.StatusOK, "detalle del evento")

		var envoltura struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltura); err != nil {
			t.Fatalf("el detalle deberia ser un objeto: %v — cuerpo: %s", err, respuesta.Body.String())
		}
		if envoltura.Data["Name"] == nil {
			t.Error("el objeto del detalle deberia traer el nombre del evento")
		}
	})
}

// ── Torneos ────────────────────────────────────────────────────────────────

func TestTorneosPorHTTP(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el cuerpo exacto que envia el formulario de torneos, con los nombres de clave del cliente")

	a := nuevaAPI(t)
	testutil.SilenciarLogs(t)

	// crearPartido devuelve el ID de un partido nuevo de la disciplina 1, creado
	// por la API igual que lo haria el formulario de partidos.
	crearPartido := func(t *testing.T, nombre, fecha string) float64 {
		t.Helper()

		respuesta := a.llamar(t, "POST", "/events/create", a.token, map[string]any{
			"Name":                 nombre,
			"Date":                 fecha,
			"Status":               "Pendiente",
			"HomeTeamID":           a.cat.EquipoLocalID,
			"OppositeTeamID":       a.cat.EquipoVisitID,
			"ResponsableTeacherID": a.cat.ProfesorID,
			"DisciplineID":         a.cat.Disciplina1ID,
		})
		testutil.Igual(t, respuesta.Code, http.StatusOK, "crear el partido")

		var envoltura struct {
			Data map[string]any `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer el partido creado")
		return envoltura.Data["ID"].(float64)
	}

	// torneoDelPartido devuelve el torneo con el que quedo el partido, o 0.
	torneoDelPartido := func(t *testing.T, partidoID float64) uint {
		t.Helper()

		var torneoID uint
		testutil.SinError(t,
			a.db.Raw("SELECT COALESCE(tourney_id, 0) FROM events WHERE id = ?", uint(partidoID)).Scan(&torneoID).Error,
			"consultar el torneo del partido")
		return torneoID
	}

	t.Run("crear con EventIDs asocia los partidos", func(t *testing.T) {
		// La clave es la que escribe el cliente. Cuando el DTO esperaba otro nombre,
		// la peticion respondia 200 y el torneo quedaba sin partidos.
		primero := crearPartido(t, "Semifinal", "2026-08-10T10:00:00Z")
		segundo := crearPartido(t, "Final", "2026-08-20T10:00:00Z")

		respuesta := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":         "Copa UNEG 2026",
			"Status":       "Pendiente",
			"DisciplineID": a.cat.Disciplina1ID,
			"StartDate":    "2026-08-10T00:00:00Z",
			"EndDate":      "2026-08-20T00:00:00Z",
			"EventIDs":     []float64{primero, segundo},
		})
		testutil.Igual(t, respuesta.Code, http.StatusOK, "crear el torneo")

		var envoltura struct {
			Data map[string]any `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer el torneo creado")
		torneoID := uint(envoltura.Data["ID"].(float64))

		testutil.Igual(t, torneoDelPartido(t, primero), torneoID, "torneo del primer partido")
		testutil.Igual(t, torneoDelPartido(t, segundo), torneoID, "torneo del segundo partido")
	})

	t.Run("sin disciplina responde 400", func(t *testing.T) {
		respuesta := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":   "Copa sin disciplina",
			"Status": "Pendiente",
		})
		testutil.Igual(t, respuesta.Code, http.StatusBadRequest, "torneo sin disciplina")
	})

	t.Run("con el fin antes del inicio responde 400", func(t *testing.T) {
		respuesta := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":         "Copa al reves",
			"Status":       "Pendiente",
			"DisciplineID": a.cat.Disciplina1ID,
			"StartDate":    "2026-08-20T00:00:00Z",
			"EndDate":      "2026-08-10T00:00:00Z",
		})
		testutil.Igual(t, respuesta.Code, http.StatusBadRequest, "rango de fechas invertido")
	})

	t.Run("el detalle trae la disciplina y los partidos que precarga el formulario", func(t *testing.T) {
		partido := crearPartido(t, "Repechaje", "2026-09-01T10:00:00Z")

		crear := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":         "Copa con detalle",
			"Status":       "Activo",
			"DisciplineID": a.cat.Disciplina1ID,
			"EventIDs":     []float64{partido},
		})
		testutil.Igual(t, crear.Code, http.StatusOK, "crear el torneo")

		var creado struct {
			Data map[string]any `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(crear.Body.Bytes(), &creado), "leer el torneo creado")
		id := strconv.FormatUint(uint64(creado.Data["ID"].(float64)), 10)

		respuesta := a.llamar(t, "GET", "/tourneys/"+id, a.token, nil)
		testutil.Igual(t, respuesta.Code, http.StatusOK, "detalle del torneo")

		var envoltura struct {
			Data map[string]any `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer el detalle")

		disciplinaID, _ := envoltura.Data["DisciplineID"].(float64)
		disciplina, _ := envoltura.Data["DisciplineName"].(string)
		partidos, _ := envoltura.Data["Events"].([]any)
		testutil.Igual(t, disciplinaID, float64(a.cat.Disciplina1ID), "disciplina del torneo")
		testutil.Igual(t, disciplina, "Voleibol", "nombre de la disciplina")
		testutil.Igual(t, len(partidos), 1, "partidos en el detalle")
	})

	t.Run("editar con la lista vacia deja el torneo sin partidos", func(t *testing.T) {
		partido := crearPartido(t, "Cuartos", "2026-09-05T10:00:00Z")

		crear := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":         "Copa para vaciar",
			"Status":       "Activo",
			"DisciplineID": a.cat.Disciplina1ID,
			"EventIDs":     []float64{partido},
		})
		testutil.Igual(t, crear.Code, http.StatusOK, "crear el torneo")

		var creado struct {
			Data map[string]any `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(crear.Body.Bytes(), &creado), "leer el torneo creado")
		id := strconv.FormatUint(uint64(creado.Data["ID"].(float64)), 10)

		editar := a.llamar(t, "PUT", "/tourneys/edit/"+id, a.token, map[string]any{
			"Name":     "Copa para vaciar",
			"EventIDs": []float64{},
		})
		testutil.Igual(t, editar.Code, http.StatusOK, "editar el torneo")
		testutil.Igual(t, torneoDelPartido(t, partido), uint(0), "torneo del partido tras vaciar la lista")
	})

	t.Run("editar un partido sin mencionar el torneo lo conserva", func(t *testing.T) {
		// Es el caso del formulario de partidos guardando solo el marcador.
		partido := crearPartido(t, "Octavos", "2026-09-08T10:00:00Z")

		crear := a.llamar(t, "POST", "/tourneys/create", a.token, map[string]any{
			"Name":         "Copa que conserva",
			"Status":       "Activo",
			"DisciplineID": a.cat.Disciplina1ID,
			"EventIDs":     []float64{partido},
		})
		testutil.Igual(t, crear.Code, http.StatusOK, "crear el torneo")

		antes := torneoDelPartido(t, partido)
		if antes == 0 {
			t.Fatal("el partido deberia haber quedado asociado al torneo")
		}

		editar := a.llamar(t, "PUT", "/events/edit/"+strconv.FormatUint(uint64(partido), 10), a.token, map[string]any{
			"Name":                 "Octavos",
			"Date":                 "2026-09-08T10:00:00Z",
			"Status":               "Finalizado",
			"HomePoints":           3,
			"OppositePoints":       1,
			"HomeTeamID":           a.cat.EquipoLocalID,
			"OppositeTeamID":       a.cat.EquipoVisitID,
			"ResponsableTeacherID": a.cat.ProfesorID,
			"DisciplineID":         a.cat.Disciplina1ID,
		})
		testutil.Igual(t, editar.Code, http.StatusOK, "editar el marcador del partido")
		testutil.Igual(t, torneoDelPartido(t, partido), antes, "torneo del partido tras editar el marcador")
	})
}

// ── Cambio de contrasena ───────────────────────────────────────────────────

func TestCambioDeContrasenaPorHTTP(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el flujo completo del cambio de contrasena por la API")

	a := nuevaAPI(t)
	testutil.SilenciarLogs(t)

	t.Run("sin token responde 401", func(t *testing.T) {
		respuesta := a.llamar(t, "PUT", "/users/change-password", "", map[string]string{
			"oldPassword": "clave-de-prueba", "newPassword": "clave-nueva-123",
		})
		testutil.Igual(t, respuesta.Code, http.StatusUnauthorized, "sin token")
	})

	t.Run("con la contrasena actual incorrecta responde 400", func(t *testing.T) {
		respuesta := a.llamar(t, "PUT", "/users/change-password", a.token, map[string]string{
			"oldPassword": "equivocada", "newPassword": "clave-nueva-123",
		})
		testutil.Igual(t, respuesta.Code, http.StatusBadRequest, "contrasena actual incorrecta")
	})

	t.Run("con una contrasena nueva demasiado corta responde 400", func(t *testing.T) {
		respuesta := a.llamar(t, "PUT", "/users/change-password", a.token, map[string]string{
			"oldPassword": "clave-de-prueba", "newPassword": "corta",
		})
		testutil.Igual(t, respuesta.Code, http.StatusBadRequest, "contrasena nueva corta")
	})

	t.Run("con datos correctos cambia la contrasena", func(t *testing.T) {
		respuesta := a.llamar(t, "PUT", "/users/change-password", a.token, map[string]string{
			"oldPassword": "clave-de-prueba", "newPassword": "clave-nueva-123",
		})
		testutil.Igual(t, respuesta.Code, http.StatusOK, "cambiar la contrasena")

		// La nueva sirve para iniciar sesion y la vieja ya no.
		a.iniciarSesion(t, "admin", "clave-nueva-123")

		fallido := a.llamar(t, "POST", "/users/login", "", map[string]string{
			"username": "admin", "password": "clave-de-prueba",
		})
		testutil.Igual(t, fallido.Code, http.StatusUnauthorized, "la contrasena vieja ya no vale")
	})
}

// ── Inicio de sesion ───────────────────────────────────────────────────────

func TestInicioDeSesionPorHTTP(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "los codigos de respuesta de cada combinacion de credenciales")

	a := nuevaAPI(t)
	testutil.SilenciarLogs(t)

	casos := []struct {
		nombre   string
		cuerpo   any
		esperado int
	}{
		{"credenciales correctas", map[string]string{"username": "admin", "password": "clave-de-prueba"}, http.StatusOK},
		{"contrasena incorrecta", map[string]string{"username": "admin", "password": "mala"}, http.StatusUnauthorized},
		{"usuario inexistente", map[string]string{"username": "fantasma", "password": "x"}, http.StatusUnauthorized},
		{"sin contrasena", map[string]string{"username": "admin"}, http.StatusBadRequest},
		{"cuerpo vacio", map[string]string{}, http.StatusBadRequest},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := a.llamar(t, "POST", "/users/login", "", caso.cuerpo)
			testutil.Igual(t, respuesta.Code, caso.esperado, "codigo de respuesta")
		})
	}
}

// idTexto convierte un identificador en la cadena que se usa en la URL.
func idTexto(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// ── Perfil de la cuenta ─────────────────────────────────────────────────────

func TestPerfilHTTP(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "las rutas del perfil como las usa el cliente, incluido que exigen token")

	t.Run("sin token no se puede leer ni cambiar el perfil", func(t *testing.T) {
		a := nuevaAPI(t)
		testutil.SilenciarLogs(t)

		testutil.Igual(t, a.llamar(t, "GET", "/users/me", "", nil).Code,
			http.StatusUnauthorized, "GET /users/me sin token")
		testutil.Igual(t, a.llamar(t, "PUT", "/users/change-username", "", map[string]string{
			"currentPassword": "clave-de-prueba", "newUsername": "otro@uneg.edu.ve",
		}).Code, http.StatusUnauthorized, "PUT /users/change-username sin token")
	})

	t.Run("devuelve el correo de la sesion y no la contrasena", func(t *testing.T) {
		a := nuevaAPI(t)

		respuesta := a.llamar(t, "GET", "/users/me", a.token, nil)
		testutil.Igual(t, respuesta.Code, http.StatusOK, "GET /users/me")

		var envoltura struct {
			Data schema.UserProfileDTO `json:"data"`
		}
		testutil.SinError(t, json.Unmarshal(respuesta.Body.Bytes(), &envoltura), "leer el perfil")
		testutil.Igual(t, envoltura.Data.Username, "admin", "correo del perfil")
		if strings.Contains(strings.ToLower(respuesta.Body.String()), "password") {
			t.Error("la respuesta del perfil no deberia mencionar la contrasena")
		}
	})

	t.Run("cambia el correo y el token abierto sigue valiendo", func(t *testing.T) {
		// El token identifica por id, asi que cambiar el correo no obliga a volver a
		// entrar: si lo hiciera, el cliente perderia la sesion en mitad del formulario.
		a := nuevaAPI(t)

		// Con espacios y mayusculas: el correo se normaliza antes de validarlo, asi
		// que pegarlo desde el gestor de contrasenas no lo convierte en invalido.
		respuesta := a.llamar(t, "PUT", "/users/change-username", a.token, map[string]string{
			"currentPassword": "clave-de-prueba", "newUsername": "  Nuevo@UNEG.edu.ve ",
		})
		testutil.Igual(t, respuesta.Code, http.StatusOK, "cambiar el correo")

		testutil.Igual(t, a.llamar(t, "GET", "/athletes", a.token, nil).Code,
			http.StatusOK, "el token de antes del cambio")
		a.iniciarSesion(t, "nuevo@uneg.edu.ve", "clave-de-prueba") // falla la prueba si no entra
	})

	t.Run("rechaza los datos invalidos", func(t *testing.T) {
		a := nuevaAPI(t)
		testutil.SilenciarLogs(t)

		casos := []struct {
			nombre   string
			cuerpo   map[string]string
			esperado int
		}{
			{"contrasena actual incorrecta", map[string]string{
				"currentPassword": "mala", "newUsername": "otro@uneg.edu.ve"}, http.StatusBadRequest},
			{"correo sin formato", map[string]string{
				"currentPassword": "clave-de-prueba", "newUsername": "no-es-un-correo"}, http.StatusBadRequest},
			{"sin contrasena actual", map[string]string{
				"newUsername": "otro@uneg.edu.ve"}, http.StatusBadRequest},
			{"cuerpo vacio", map[string]string{}, http.StatusBadRequest},
		}

		for _, caso := range casos {
			t.Run(caso.nombre, func(t *testing.T) {
				respuesta := a.llamar(t, "PUT", "/users/change-username", a.token, caso.cuerpo)
				testutil.Igual(t, respuesta.Code, caso.esperado, "codigo de respuesta")
			})
		}

		// Ninguno de los rechazos debio tocar la credencial.
		a.iniciarSesion(t, "admin", "clave-de-prueba")
	})

	t.Run("cambia la contrasena y el correo se conserva", func(t *testing.T) {
		a := nuevaAPI(t)

		respuesta := a.llamar(t, "PUT", "/users/change-password", a.token, map[string]string{
			"oldPassword": "clave-de-prueba", "newPassword": "clave-nueva-larga",
		})
		testutil.Igual(t, respuesta.Code, http.StatusOK, "cambiar la contrasena")

		a.iniciarSesion(t, "admin", "clave-nueva-larga") // falla la prueba si no entra
	})
}
