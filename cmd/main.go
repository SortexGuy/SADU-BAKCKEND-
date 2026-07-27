package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/internal/handlers"
	"uneg.edu.ve/servicio-sadu-back/internal/logging"
	"uneg.edu.ve/servicio-sadu-back/internal/middlewares"
	"uneg.edu.ve/servicio-sadu-back/internal/routes"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
)

func main() {
	// El registro se configura antes que nada, para que cualquier problema de
	// arranque quede escrito con el mismo formato que el resto.
	logging.Setup()

	config.LoadEnv()
	if len(config.SecretKey()) == 0 {
		slog.Error("SECRET_KEY no esta definida: el servidor no arranca sin ella. " +
			"Configurala en el archivo .env o en las variables de entorno del despliegue.")
		os.Exit(1)
	}

	config.ConnectDB()
	if err := config.SyncDB(); err != nil {
		slog.Error("no se pudo preparar el esquema de la base de datos", "error", err.Error())
		os.Exit(1)
	}

	db := config.DB
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	athleteService := services.AthleteService{DB: db}
	athleteHandler := handlers.NewAthleteHandler(&athleteService)

	universityService := services.UniversityServices{DB: db}
	universityHandler := handlers.NewUniversityHandler(&universityService)

	disciplineService := services.DisciplineServices{DB: db}
	disciplineHandler := handlers.NewDisciplineHandler(&disciplineService)

	majorService := services.MajorServices{DB: db}
	majorHandler := handlers.NewMajorHandler(&majorService)

	tourneyService := services.TourneyServices{DB: db}
	tourneyHandler := handlers.NewTourneyHandler(&tourneyService)

	teacherService := services.TeacherService{DB: db}
	teacherHandler := handlers.NewTeacherHandler(&teacherService)

	teamService := services.TeamServices{DB: db}
	teamHandler := handlers.NewTeamHandler(&teamService)

	eventService := services.EventService{DB: db}
	eventHandlers := handlers.NewEventHandler(&eventService)

	userService := services.UserService{DB: db}
	if err := userService.EnsureAdminUser(); err != nil {
		slog.Error("no se pudo garantizar el usuario administrador", "error", err.Error())
	}
	userHandlers := handlers.NewUserHandler(&userService)

	// Lo que Gin escribe por su cuenta (volcado de rutas y advertencias) se enruta
	// al registro con nivel de depuracion, para que no se mezcle con el flujo
	// estructurado en operacion normal.
	gin.DefaultWriter = logging.EscritorDepuracion()
	gin.DefaultErrorWriter = logging.EscritorDepuracion()
	gin.DebugPrintRouteFunc = func(metodo, ruta, handler string, cantidad int) {
		slog.Debug("ruta registrada", "metodo", metodo, "ruta", ruta, "handlers", cantidad)
	}

	// gin.New() en lugar de gin.Default(): el registro de acceso y la recuperacion
	// de panics los aportan los middlewares propios, que escriben estructurado.
	r := gin.New()
	r.Use(middlewares.RequestID(), middlewares.AccessLog(), middlewares.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	//configuracion de CORS
	// Solo se agrega CORS_DOMAIN si tiene valor: una cadena vacia en la lista de
	// origenes hace que gin-contrib/cors entre en panic al arrancar.
	origins := []string{"http://localhost:3000"}
	if domain := os.Getenv("CORS_DOMAIN"); domain != "" {
		origins = append(origins, domain)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	/*rutas*/
	routes.RegisterAthletesRoutes(r.Group("/athletes", middlewares.AuthMiddleware()), athleteHandler)
	routes.RegisterUniversityRoutes(r.Group("/universities", middlewares.AuthMiddleware()), universityHandler)
	routes.RegisterDisciplines(r.Group("/disciplines", middlewares.AuthMiddleware()), disciplineHandler)
	routes.RegisterMajorsRoutes(r.Group("/majors", middlewares.AuthMiddleware()), majorHandler)
	routes.RegisterTourney(r.Group("/tourneys", middlewares.AuthMiddleware()), tourneyHandler)
	routes.RegisterTeacherRoutes(r.Group("/teachers", middlewares.AuthMiddleware()), teacherHandler)
	routes.RegisterTeamRoutes(r.Group("/teams", middlewares.AuthMiddleware()), teamHandler)
	routes.RegisterEventsRouters(r.Group("/events"), eventHandlers)
	routes.RegisterUserRoutes(r.Group("/users"), userHandlers)

	// Resumen de la configuracion efectiva, sin secretos: es lo primero que hay
	// que mirar cuando el servidor arranca pero no se comporta como se esperaba.
	slog.Info("servidor iniciado",
		"puerto", port,
		"url", "http://localhost:"+port,
		"modo_gin", gin.Mode(),
		"origenes_cors", origins,
		"rutas", len(r.Routes()),
	)

	if err := r.Run(":" + port); err != nil {
		slog.Error("el servidor se detuvo", "error", err.Error())
		os.Exit(1)
	}
}
