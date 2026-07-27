package services_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// contextoConID arma un *gin.Context con el parametro :id, porque varios metodos
// del servicio leen el identificador de la ruta en lugar de recibirlo.
func contextoConID(id string) *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Params = gin.Params{{Key: "id", Value: id}}
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	return ctx
}

func atletaValido(cat testutil.Catalogo, cedula string) schema.Athlete {
	return schema.Athlete{
		FirstNames: "Ana",
		LastNames:  "Perez",
		GovID:      cedula,
		Gender:     schema.GenderF,
		Enrolled:   true,
		MajorID:    cat.CarreraID,
	}
}

func TestCrearAtleta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "ataca la transaccion del alta, las claves foraneas y las tablas puente")

	t.Run("con datos completos queda registrado", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		creado, err := servicio.CreateAthlete(atletaValido(cat, "30000001"))
		testutil.SinError(t, err, "crear atleta")

		if creado.ID == 0 {
			t.Fatal("el atleta creado no recibio identificador")
		}

		var guardado schema.Athlete
		testutil.SinError(t, db.First(&guardado, creado.ID).Error, "leer el atleta creado")
		testutil.Igual(t, guardado.GovID, "30000001", "cedula guardada")
		testutil.Igual(t, guardado.MajorID, cat.CarreraID, "carrera guardada")
	})

	t.Run("sin cedula se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		_, err := servicio.CreateAthlete(atletaValido(cat, "   "))
		if !errors.Is(err, services.ErrMissingGovID) {
			t.Fatalf("se esperaba ErrMissingGovID y se obtuvo: %v", err)
		}
	})

	t.Run("sin carrera se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		atleta := schema.Athlete{FirstNames: "Sin", LastNames: "Carrera", GovID: "30000002"}
		_, err := servicio.CreateAthlete(atleta)
		if !errors.Is(err, services.ErrMissingMajor) {
			t.Fatalf("se esperaba ErrMissingMajor y se obtuvo: %v", err)
		}
	})

	t.Run("con cedula repetida se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		_, err := servicio.CreateAthlete(atletaValido(cat, "30000003"))
		testutil.SinError(t, err, "crear el primer atleta")

		otro := atletaValido(cat, "30000003")
		otro.FirstNames = "Otro"
		_, err = servicio.CreateAthlete(otro)
		if !errors.Is(err, services.ErrDuplicateGovID) {
			t.Fatalf("se esperaba ErrDuplicateGovID y se obtuvo: %v", err)
		}

		var total int64
		db.Model(&schema.Athlete{}).Count(&total)
		testutil.Igual(t, total, int64(1), "atletas guardados tras el rechazo")
	})

	t.Run("con carrera inexistente lo rechaza la clave foranea", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.AthleteService{DB: db}

		atleta := schema.Athlete{FirstNames: "X", LastNames: "Y", GovID: "30000004", MajorID: 999}
		_, err := servicio.CreateAthlete(atleta)
		if err == nil {
			t.Fatal("se esperaba un error de referencia invalida")
		}
		if !services.IsInvalidReference(err) {
			t.Fatalf("el error deberia reconocerse como referencia invalida: %v", err)
		}
	})

	t.Run("una vinculacion invalida no deja el atleta a medio crear", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.AthleteService{DB: db}

		atleta := atletaValido(cat, "30000005")
		atleta.Teams = []schema.Team{{Model: gorm.Model{ID: 999}}} // equipo inexistente

		_, err := servicio.CreateAthlete(atleta)
		if err == nil {
			t.Fatal("se esperaba error al vincular un equipo inexistente")
		}

		// La transaccion debe haber revertido el alta completa.
		var total int64
		db.Model(&schema.Athlete{}).Count(&total)
		testutil.Igual(t, total, int64(0), "atletas guardados tras la reversion")
	})

	t.Run("guarda las vinculaciones validas", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		atleta := atletaValido(cat, "30000006")
		atleta.Disciplines = []schema.Discipline{{Model: gorm.Model{ID: uint(cat.Disciplina1ID)}}}
		atleta.Teams = []schema.Team{{Model: gorm.Model{ID: uint(cat.EquipoLocalID)}}}

		creado, err := servicio.CreateAthlete(atleta)
		testutil.SinError(t, err, "crear atleta con vinculaciones")

		var disciplinas, equipos int64
		db.Table("athlete_disciplines").Where("athlete_id = ?", creado.ID).Count(&disciplinas)
		db.Table("athlete_teams").Where("athlete_id = ?", creado.ID).Count(&equipos)
		testutil.Igual(t, disciplinas, int64(1), "disciplinas vinculadas")
		testutil.Igual(t, equipos, int64(1), "equipos vinculados")
	})
}

func TestEditarAtleta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "ataca la exclusion del propio registro al comprobar la cedula")

	t.Run("permite conservar la propia cedula", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		creado, err := servicio.CreateAthlete(atletaValido(cat, "31000001"))
		testutil.SinError(t, err, "crear atleta")

		cambios := schema.Athlete{FirstNames: "Ana Maria", GovID: "31000001"}
		editado, err := servicio.EditAthlete(cambios, contextoConID(idComoTexto(creado.ID)))
		testutil.SinError(t, err, "editar conservando la cedula")
		testutil.Igual(t, editado.FirstNames, "Ana Maria", "nombre actualizado")
	})

	t.Run("rechaza tomar la cedula de otro atleta", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		primero, err := servicio.CreateAthlete(atletaValido(cat, "31000002"))
		testutil.SinError(t, err, "crear el primer atleta")
		segundo, err := servicio.CreateAthlete(atletaValido(cat, "31000003"))
		testutil.SinError(t, err, "crear el segundo atleta")

		_, err = servicio.EditAthlete(
			schema.Athlete{GovID: primero.GovID},
			contextoConID(idComoTexto(segundo.ID)),
		)
		if !errors.Is(err, services.ErrDuplicateGovID) {
			t.Fatalf("se esperaba ErrDuplicateGovID y se obtuvo: %v", err)
		}
	})

	t.Run("un atleta inexistente da error", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.AthleteService{DB: db}

		_, err := servicio.EditAthlete(schema.Athlete{FirstNames: "X"}, contextoConID("999"))
		if err == nil {
			t.Fatal("se esperaba error al editar un atleta inexistente")
		}
	})
}

func TestEliminarAtleta(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "verifica el borrado logico con Unscoped y la rama del id no numerico")

	t.Run("elimina de forma logica y lo saca de las consultas", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		creado, err := servicio.CreateAthlete(atletaValido(cat, "32000001"))
		testutil.SinError(t, err, "crear atleta")

		testutil.SinError(t, servicio.DeleteAthlete(contextoConID(idComoTexto(creado.ID))), "eliminar atleta")

		// Fuera de las consultas normales...
		var visibles int64
		db.Model(&schema.Athlete{}).Count(&visibles)
		testutil.Igual(t, visibles, int64(0), "atletas visibles tras eliminar")

		// ...pero la fila sigue en la tabla, con deleted_at.
		var conBorrados int64
		db.Unscoped().Model(&schema.Athlete{}).Count(&conBorrados)
		testutil.Igual(t, conBorrados, int64(1), "filas conservadas por el borrado logico")
	})

	t.Run("un atleta inexistente da error", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.AthleteService{DB: db}

		if err := servicio.DeleteAthlete(contextoConID("999")); err == nil {
			t.Fatal("eliminar un atleta inexistente deberia dar error, no exito")
		}
	})

	t.Run("un identificador no numerico da error", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.AthleteService{DB: db}

		if err := servicio.DeleteAthlete(contextoConID("abc")); err == nil {
			t.Fatal("un identificador invalido deberia dar error")
		}
	})
}

func TestFiltrosDeAtletas(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre una rama por cada filtro del constructor de la consulta")

	// El buscador unico coincide con nombre, apellido o cedula; los demas filtros
	// se combinan con AND. Estas pruebas cubren las dos semanticas.
	preparar := func(t *testing.T) (*gorm.DB, testutil.Catalogo) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.AthleteService{DB: db}

		ana := atletaValido(cat, "40000001")
		ana.FirstNames, ana.LastNames, ana.Gender = "Ana", "Perez", schema.GenderF
		ana.Disciplines = []schema.Discipline{{Model: gorm.Model{ID: uint(cat.Disciplina1ID)}}}
		_, err := servicio.CreateAthlete(ana)
		testutil.SinError(t, err, "crear a Ana")

		beto := atletaValido(cat, "41111111")
		beto.FirstNames, beto.LastNames, beto.Gender = "Beto", "Quintero", schema.GenderM
		beto.Disciplines = []schema.Discipline{{Model: gorm.Model{ID: uint(cat.Disciplina2ID)}}}
		_, err = servicio.CreateAthlete(beto)
		testutil.SinError(t, err, "crear a Beto")

		return db, cat
	}

	casos := []struct {
		nombre                                      string
		name, lastname, govID, gender, disciplinaID string
		search                                      string
		esperados                                   int
	}{
		{nombre: "sin filtros devuelve todos", esperados: 2},
		{nombre: "search por nombre", search: "Ana", esperados: 1},
		{nombre: "search por apellido", search: "Quintero", esperados: 1},
		{nombre: "search por cedula", search: "41111111", esperados: 1},
		{nombre: "search sin coincidencias", search: "zzz", esperados: 0},
		{nombre: "search parcial coincide con ambos", search: "0000", esperados: 1},
		{nombre: "gender filtra", gender: string(schema.GenderF), esperados: 1},
		{nombre: "search y gender se combinan con AND", search: "Ana", gender: string(schema.GenderM), esperados: 0},
		{nombre: "search y gender coincidentes", search: "Ana", gender: string(schema.GenderF), esperados: 1},
		{nombre: "name filtra por nombre", name: "Beto", esperados: 1},
		{nombre: "lastname filtra por apellido", lastname: "Perez", esperados: 1},
		{nombre: "govID filtra por cedula", govID: "40000001", esperados: 1},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, _ = preparar(t)

			resultado, err := services.GetAllAthletes(
				caso.name, caso.lastname, caso.govID, caso.gender, caso.disciplinaID, caso.search,
			)
			testutil.SinError(t, err, "listar atletas")
			testutil.Igual(t, len(resultado), caso.esperados, "cantidad de atletas")
		})
	}

	t.Run("discipline_id filtra por la disciplina vinculada", func(t *testing.T) {
		_, cat := preparar(t)

		soloDisciplina1, err := services.GetAllAthletes("", "", "", "", idComoTexto(uint(cat.Disciplina1ID)), "")
		testutil.SinError(t, err, "filtrar por disciplina")
		testutil.Igual(t, len(soloDisciplina1), 1, "atletas de la disciplina 1")
		if len(soloDisciplina1) == 1 {
			testutil.Igual(t, soloDisciplina1[0].FirstNames, "Ana", "atleta de la disciplina 1")
		}
	})

	t.Run("el listado incluye las disciplinas de cada atleta", func(t *testing.T) {
		preparar(t)

		resultado, err := services.GetAllAthletes("", "", "", "", "", "Ana")
		testutil.SinError(t, err, "listar atletas")
		if len(resultado) != 1 {
			t.Fatalf("se esperaba un atleta y se obtuvieron %d", len(resultado))
		}
		testutil.Igual(t, len(resultado[0].Disciplines), 1, "disciplinas precargadas")
	})

	t.Run("un atleta eliminado no aparece", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.AthleteService{DB: db}

		var ana schema.Athlete
		testutil.SinError(t, db.Where("first_names = ?", "Ana").First(&ana).Error, "buscar a Ana")
		testutil.SinError(t, servicio.DeleteAthlete(contextoConID(idComoTexto(ana.ID))), "eliminar a Ana")

		resultado, err := services.GetAllAthletes("", "", "", "", "", "")
		testutil.SinError(t, err, "listar atletas")
		testutil.Igual(t, len(resultado), 1, "atletas visibles tras eliminar uno")
	})
}

func TestObtenerAtletaPorID(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "comprueba la precarga de relaciones, que no se ve desde la interfaz")

	db := testutil.NuevaDB(t)
	cat := testutil.SembrarCatalogo(t, db)
	testutil.SilenciarLogs(t)
	servicio := services.AthleteService{DB: db}

	atleta := atletaValido(cat, "50000001")
	atleta.Teams = []schema.Team{{Model: gorm.Model{ID: uint(cat.EquipoLocalID)}}}
	creado, err := servicio.CreateAthlete(atleta)
	testutil.SinError(t, err, "crear atleta")

	t.Run("devuelve el atleta con sus relaciones precargadas", func(t *testing.T) {
		obtenido, err := servicio.GetAthletesByID(contextoConID(idComoTexto(creado.ID)))
		testutil.SinError(t, err, "obtener atleta por id")
		testutil.Igual(t, obtenido.GovID, "50000001", "cedula")
		testutil.Igual(t, len(obtenido.Teams), 1, "equipos precargados")
	})

	t.Run("un id inexistente da error", func(t *testing.T) {
		if _, err := servicio.GetAthletesByID(contextoConID("999")); err == nil {
			t.Fatal("se esperaba error con un id inexistente")
		}
	})

	t.Run("un id no numerico da error", func(t *testing.T) {
		if _, err := servicio.GetAthletesByID(contextoConID("abc")); err == nil {
			t.Fatal("se esperaba error con un id no numerico")
		}
	})
}
