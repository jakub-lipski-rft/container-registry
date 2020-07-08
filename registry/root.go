package registry

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/migrations"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver/factory"
	"github.com/docker/distribution/version"

	"github.com/docker/libtrust"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var showVersion bool
var maxNumMigrations *int

func init() {
	RootCmd.AddCommand(ServeCmd)
	RootCmd.AddCommand(GCCmd)
	RootCmd.AddCommand(DBCmd)
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show the version and exit")

	GCCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do everything except remove the blobs")
	GCCmd.Flags().BoolVarP(&removeUntagged, "delete-untagged", "m", false, "delete manifests that are not currently referenced via tag")
	GCCmd.Flags().StringVarP(&debugAddr, "debug-server", "s", "", "run a pprof debug server at <address:port>")

	MigrateCmd.AddCommand(MigrateVersionCmd)
	MigrateUpCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not commit changes to the database")
	MigrateUpCmd.Flags().VarP(nullableInt{&maxNumMigrations}, "limit", "n", "limit the number of migrations (all by default)")
	MigrateCmd.AddCommand(MigrateUpCmd)
	MigrateDownCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not commit changes to the database")
	MigrateDownCmd.Flags().VarP(nullableInt{&maxNumMigrations}, "limit", "n", "limit the number of migrations (all by default)")
	MigrateCmd.AddCommand(MigrateDownCmd)
	DBCmd.AddCommand(MigrateCmd)

	DBCmd.AddCommand(ImportCmd)
	ImportCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not commit changes to the database")
}

// nullableInt implements spf13/pflag#Value as a custom nullable integer to capture spf13/cobra command flags.
// https://pkg.go.dev/github.com/spf13/pflag?tab=doc#Value
type nullableInt struct {
	ptr **int
}

func (f nullableInt) String() string {
	if *f.ptr == nil {
		return "0"
	}
	return strconv.Itoa(**f.ptr)
}

func (f nullableInt) Type() string {
	return "int"
}

func (f nullableInt) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*f.ptr = &v
	return nil
}

// RootCmd is the main command for the 'registry' binary.
var RootCmd = &cobra.Command{
	Use:   "registry",
	Short: "`registry`",
	Long:  "`registry`",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			version.PrintVersion()
			return
		}
		cmd.Usage()
	},
}

var dryRun bool
var removeUntagged bool
var debugAddr string

// GCCmd is the cobra command that corresponds to the garbage-collect subcommand
var GCCmd = &cobra.Command{
	Use:   "garbage-collect <config>",
	Short: "`garbage-collect` deletes layers not referenced by any manifests",
	Long:  "`garbage-collect` deletes layers not referenced by any manifests",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := resolveConfiguration(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		driver, err := factory.Create(config.Storage.Type(), config.Storage.Parameters())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct %s driver: %v", config.Storage.Type(), err)
			os.Exit(1)
		}

		ctx := dcontext.Background()
		ctx, err = configureLogging(ctx, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to configure logging with config: %s", err)
			os.Exit(1)
		}

		k, err := libtrust.GenerateECP256PrivateKey()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		registry, err := storage.NewRegistry(ctx, driver, storage.Schema1SigningKey(k))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct registry: %v", err)
			os.Exit(1)
		}

		if debugAddr != "" {
			go func() {
				dcontext.GetLoggerWithField(ctx, "address", debugAddr).Info("debug server listening")
				if err := http.ListenAndServe(debugAddr, nil); err != nil {
					dcontext.GetLoggerWithField(ctx, "error", err).Fatal("error listening on debug interface")
				}
			}()
		}

		err = storage.MarkAndSweep(ctx, driver, registry, storage.GCOpts{
			DryRun:         dryRun,
			RemoveUntagged: removeUntagged,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to garbage collect: %v", err)
			os.Exit(1)
		}
	},
}

// DBCmd is the root of the `database` command.
var DBCmd = &cobra.Command{
	Use:   "database",
	Short: "Manages the registry metadata database",
	Long:  "Manages the registry metadata database",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

// MigrateCmd is the `migrate` sub-command of `database` that manages database migrations.
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage migrations",
	Long:  "Manage migrations",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var MigrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply up migrations",
	Long:  "Apply up migrations",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := resolveConfiguration(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		if maxNumMigrations == nil {
			var all int
			maxNumMigrations = &all
		} else if *maxNumMigrations < 1 {
			fmt.Fprintf(os.Stderr, "limit must be greater than or equal to 1")
			os.Exit(1)
		}

		db, err := datastore.Open(&datastore.DSN{
			Host:     config.Database.Host,
			Port:     config.Database.Port,
			User:     config.Database.User,
			Password: config.Database.Password,
			DBName:   config.Database.DBName,
			SSLMode:  config.Database.SSLMode,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct database connection: %v", err)
			os.Exit(1)
		}

		m := migrations.NewMigrator(db.DB)
		plan, err := m.UpNPlan(*maxNumMigrations)
		if !dryRun {
			if err := m.UpN(*maxNumMigrations); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run database migrations: %v", err)
				os.Exit(1)
			}
		}
		fmt.Println(strings.Join(plan, "\n"))
	},
}

var MigrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Apply down migrations",
	Long:  "Apply down migrations",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := resolveConfiguration(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		if maxNumMigrations == nil {
			var all int
			maxNumMigrations = &all
		} else if *maxNumMigrations < 1 {
			fmt.Fprintf(os.Stderr, "limit must be greater than or equal to 1")
			os.Exit(1)
		}

		db, err := datastore.Open(&datastore.DSN{
			Host:     config.Database.Host,
			Port:     config.Database.Port,
			User:     config.Database.User,
			Password: config.Database.Password,
			DBName:   config.Database.DBName,
			SSLMode:  config.Database.SSLMode,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct database connection: %v", err)
			os.Exit(1)
		}

		m := migrations.NewMigrator(db.DB)
		plan, err := m.DownNPlan(*maxNumMigrations)
		if !dryRun {
			if err := m.DownN(*maxNumMigrations); err != nil {
				fmt.Fprintf(os.Stderr, "failed to run database migrations: %v", err)
				os.Exit(1)
			}
		}
		fmt.Println(strings.Join(plan, "\n"))
	},
}

// MigrateVersionCmd is the `version` sub-command of `database migrate` that shows the current migration version.
var MigrateVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current migration version",
	Long:  "Show current migration version",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := resolveConfiguration(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		db, err := datastore.Open(&datastore.DSN{
			Host:     config.Database.Host,
			Port:     config.Database.Port,
			User:     config.Database.User,
			Password: config.Database.Password,
			DBName:   config.Database.DBName,
			SSLMode:  config.Database.SSLMode,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct database connection: %v", err)
			os.Exit(1)
		}

		m := migrations.NewMigrator(db.DB)
		v, err := m.Version()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to detect database version: %v", err)
			os.Exit(1)
		}
		if v == "" {
			v = "Unknown"
		}

		fmt.Printf("%s\n", v)
	},
}

// ImportCmd is the `import` sub-command of `database` that imports metadata from the filesystem into the database.
var ImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import filesystem metadata into the database",
	Long: "Import filesystem metadata into the database. This should only be\n" +
		"used for an one-off migration, starting with an empty database.\n" +
		"Dangling blobs are not imported. This tool can not be used with\n" +
		"the parallelwalk storage configuration enabled.",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := resolveConfiguration(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		parameters := config.Storage.Parameters()
		if parameters["parallelwalk"] == true {
			parameters["parallelwalk"] = false
			logrus.Info("the 'parallelwalk' configuration parameter has been disabled")
		}

		driver, err := factory.Create(config.Storage.Type(), parameters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct %s driver: %v", config.Storage.Type(), err)
			os.Exit(1)
		}

		ctx := dcontext.Background()
		ctx, err = configureLogging(ctx, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to configure logging with config: %s", err)
			os.Exit(1)
		}

		k, err := libtrust.GenerateECP256PrivateKey()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		registry, err := storage.NewRegistry(ctx, driver, storage.Schema1SigningKey(k))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct registry: %v", err)
			os.Exit(1)
		}

		db, err := datastore.Open(&datastore.DSN{
			Host:     config.Database.Host,
			Port:     config.Database.Port,
			User:     config.Database.User,
			Password: config.Database.Password,
			DBName:   config.Database.DBName,
			SSLMode:  config.Database.SSLMode,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to construct database connection: %v", err)
			os.Exit(1)
		}

		tx, err := db.Begin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create database transaction: %v", err)
			os.Exit(1)
		}

		// recover from panic, to rollback transaction and re-panic
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("recovered from panic, rolling back changes", r)
				if err := tx.Rollback(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to rollback changes: %v", err)
				}
				panic(r)
			}
		}()

		p := datastore.NewImporter(tx, driver, registry)
		if err := p.Import(ctx); err != nil {
			tx.Rollback()
			fmt.Fprintf(os.Stderr, "failed to import metadata: %v", err)
			os.Exit(1)
		}

		if dryRun {
			if err := tx.Rollback(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to rollback changes: %v", err)
				os.Exit(1)
			}
		} else {
			if err := tx.Commit(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to commit changes: %v", err)
				os.Exit(1)
			}
		}
	},
}
