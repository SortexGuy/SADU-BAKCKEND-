package services_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

func TestCrearProfesor(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "validaciones de la especificacion; una asercion cuenta la tabla puente")

	t.Run("con datos completos queda registrado", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.TeacherService{DB: db}

		creado, err := servicio.CreateTeacher(schema.Teacher{
			FirstNames: "Marta", LastNames: "Solis", GovID: "21000001",
		})
		testutil.SinError(t, err, "crear profesor")
		if creado.ID == 0 {
			t.Fatal("el profesor creado no recibio identificador")
		}
	})

	t.Run("sin cedula se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.TeacherService{DB: db}

		_, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "Sin", LastNames: "Cedula"})
		if !errors.Is(err, services.ErrMissingGovID) {
			t.Fatalf("se esperaba ErrMissingGovID y se obtuvo: %v", err)
		}
	})

	t.Run("con cedula repetida se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.TeacherService{DB: db}

		_, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "A", LastNames: "B", GovID: "21000002"})
		testutil.SinError(t, err, "crear el primer profesor")

		_, err = servicio.CreateTeacher(schema.Teacher{FirstNames: "C", LastNames: "D", GovID: "21000002"})
		if !errors.Is(err, services.ErrDuplicateGovID) {
			t.Fatalf("se esperaba ErrDuplicateGovID y se obtuvo: %v", err)
		}

		var total int64
		db.Model(&schema.Teacher{}).Count(&total)
		testutil.Igual(t, total, int64(1), "profesores guardados")
	})

	t.Run("guarda las disciplinas vinculadas", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TeacherService{DB: db}

		creado, err := servicio.CreateTeacher(schema.Teacher{
			FirstNames: "Marta", LastNames: "Solis", GovID: "21000003",
			Disciplines: []schema.Discipline{{Model: gorm.Model{ID: uint(cat.Disciplina1ID)}}},
		})
		testutil.SinError(t, err, "crear profesor con disciplina")

		var vinculos int64
		db.Table("teacher_disciplines").Where("teacher_id = ?", creado.ID).Count(&vinculos)
		testutil.Igual(t, vinculos, int64(1), "disciplinas vinculadas")
	})
}

func TestEditarProfesor(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "la regla de unicidad de la cedula, vista desde fuera")

	t.Run("rechaza tomar la cedula de otro profesor", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.TeacherService{DB: db}

		primero, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "A", LastNames: "B", GovID: "22000001"})
		testutil.SinError(t, err, "crear el primer profesor")
		segundo, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "C", LastNames: "D", GovID: "22000002"})
		testutil.SinError(t, err, "crear el segundo profesor")

		_, err = servicio.EditTeacher(contextoConID(idComoTexto(segundo.ID)), schema.Teacher{GovID: primero.GovID})
		if !errors.Is(err, services.ErrDuplicateGovID) {
			t.Fatalf("se esperaba ErrDuplicateGovID y se obtuvo: %v", err)
		}
	})

	t.Run("permite conservar la propia cedula", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.TeacherService{DB: db}

		creado, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "A", LastNames: "B", GovID: "22000003"})
		testutil.SinError(t, err, "crear profesor")

		editado, err := servicio.EditTeacher(
			contextoConID(idComoTexto(creado.ID)),
			schema.Teacher{FirstNames: "Alberto", GovID: "22000003"},
		)
		testutil.SinError(t, err, "editar conservando la cedula")
		testutil.Igual(t, editado.FirstNames, "Alberto", "nombre actualizado")
	})
}

func TestFiltrosDeProfesores(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "recorre las ramas de los filtros y la combinacion con AND")

	preparar := func(t *testing.T) (*gorm.DB, testutil.Catalogo) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		servicio := services.TeacherService{DB: db}

		// El catalogo ya trae a Luis Ramirez sin disciplinas.
		_, err := servicio.CreateTeacher(schema.Teacher{
			FirstNames: "Marta", LastNames: "Solis", GovID: "23000001",
			Disciplines: []schema.Discipline{{Model: gorm.Model{ID: uint(cat.Disciplina1ID)}}},
		})
		testutil.SinError(t, err, "crear a Marta")
		return db, cat
	}

	t.Run("sin filtros devuelve todos", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TeacherService{DB: db}

		lista, err := servicio.GetTeachers("", "", "", "", "")
		testutil.SinError(t, err, "listar profesores")
		testutil.Igual(t, len(lista), 2, "profesores")
	})

	t.Run("el buscador coincide con nombre, apellido o cedula", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TeacherService{DB: db}

		casos := map[string]int{
			"Marta":    1, // nombre
			"Ramirez":  1, // apellido
			"23000001": 1, // cedula
			"zzz":      0,
		}
		for busqueda, esperados := range casos {
			lista, err := servicio.GetTeachers("", "", "", "", busqueda)
			testutil.SinError(t, err, "buscar profesores")
			testutil.Igual(t, len(lista), esperados, "resultados para '"+busqueda+"'")
		}
	})

	t.Run("por disciplina", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.TeacherService{DB: db}

		conDisciplina, err := servicio.GetTeachers("", "", "", idComoTexto(uint(cat.Disciplina1ID)), "")
		testutil.SinError(t, err, "filtrar por disciplina")
		testutil.Igual(t, len(conDisciplina), 1, "profesores de la disciplina 1")

		sinNadie, err := servicio.GetTeachers("", "", "", idComoTexto(uint(cat.Disciplina2ID)), "")
		testutil.SinError(t, err, "filtrar por la otra disciplina")
		testutil.Igual(t, len(sinNadie), 0, "profesores de la disciplina 2")
	})

	t.Run("el buscador y la disciplina se combinan con AND", func(t *testing.T) {
		db, cat := preparar(t)
		servicio := services.TeacherService{DB: db}

		lista, err := servicio.GetTeachers("", "", "", idComoTexto(uint(cat.Disciplina1ID)), "Ramirez")
		testutil.SinError(t, err, "combinar buscador y disciplina")
		testutil.Igual(t, len(lista), 0, "Ramirez no tiene la disciplina 1")
	})

	t.Run("el listado incluye las disciplinas de cada profesor", func(t *testing.T) {
		db, _ := preparar(t)
		servicio := services.TeacherService{DB: db}

		lista, err := servicio.GetTeachers("", "", "", "", "Marta")
		testutil.SinError(t, err, "buscar a Marta")
		if len(lista) != 1 {
			t.Fatalf("se esperaba un profesor y se obtuvieron %d", len(lista))
		}
		testutil.Igual(t, len(lista[0].Disciplines), 1, "disciplinas precargadas")
	})
}

func TestEliminarProfesor(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "existe o no existe, y el resultado esperado")

	db := testutil.NuevaDB(t)
	testutil.SilenciarLogs(t)
	servicio := services.TeacherService{DB: db}

	creado, err := servicio.CreateTeacher(schema.Teacher{FirstNames: "A", LastNames: "B", GovID: "24000001"})
	testutil.SinError(t, err, "crear profesor")

	t.Run("elimina el existente", func(t *testing.T) {
		testutil.SinError(t, servicio.DeleteTeacher(contextoConID(idComoTexto(creado.ID))), "eliminar profesor")

		var visibles int64
		db.Model(&schema.Teacher{}).Count(&visibles)
		testutil.Igual(t, visibles, int64(0), "profesores visibles")
	})

	t.Run("uno inexistente da error", func(t *testing.T) {
		if err := servicio.DeleteTeacher(contextoConID("999")); err == nil {
			t.Fatal("eliminar un profesor inexistente deberia dar error")
		}
	})
}

// Los torneos tienen su propio archivo de pruebas: tourney_services_test.go.

func TestEquipos(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "ramas de los filtros y el puntero con el que se decide filtrar por titular")

	t.Run("los filtros funcionan y regular solo filtra si se envia", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db) // Alfa: titular/Masculino/disc1, Beta: no titular/Femenino/disc2
		servicio := services.TeamServices{DB: db}

		todos, err := servicio.GetAllTeam("", "", "", "", nil)
		testutil.SinError(t, err, "listar equipos")
		testutil.Igual(t, len(todos), 2, "equipos")

		porNombre, err := servicio.GetAllTeam("Alfa", "", "", "", nil)
		testutil.SinError(t, err, "filtrar por nombre")
		testutil.Igual(t, len(porNombre), 1, "equipos con 'Alfa'")

		porCategoria, err := servicio.GetAllTeam("", string(schema.GenderF), "", "", nil)
		testutil.SinError(t, err, "filtrar por categoria")
		testutil.Igual(t, len(porCategoria), 1, "equipos femeninos")

		porDisciplina, err := servicio.GetAllTeam("", "", idComoTexto(uint(cat.Disciplina1ID)), "", nil)
		testutil.SinError(t, err, "filtrar por disciplina")
		testutil.Igual(t, len(porDisciplina), 1, "equipos de la disciplina 1")

		porUniversidad, err := servicio.GetAllTeam("", "", "", idComoTexto(uint(cat.UniversidadID)), nil)
		testutil.SinError(t, err, "filtrar por universidad")
		testutil.Igual(t, len(porUniversidad), 2, "equipos de la universidad")

		titular := true
		soloTitulares, err := servicio.GetAllTeam("", "", "", "", &titular)
		testutil.SinError(t, err, "filtrar titulares")
		testutil.Igual(t, len(soloTitulares), 1, "equipos titulares")

		noTitular := false
		soloNoTitulares, err := servicio.GetAllTeam("", "", "", "", &noTitular)
		testutil.SinError(t, err, "filtrar no titulares")
		testutil.Igual(t, len(soloNoTitulares), 1, "equipos no titulares")
	})

	t.Run("con disciplina o universidad inexistente se rechaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		cat := testutil.SembrarCatalogo(t, db)
		testutil.SilenciarLogs(t)
		servicio := services.TeamServices{DB: db}

		_, err := servicio.CreateTeam(schema.Team{
			Name: "Fantasma", DisciplineID: 999, UniversityID: cat.UniversidadID,
		})
		if err == nil {
			t.Fatal("se esperaba error con una disciplina inexistente")
		}
	})
}
