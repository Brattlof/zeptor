package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/brattlof/zeptor/internal/app/config"
	"github.com/brattlof/zeptor/internal/app/router"
	"github.com/brattlof/zeptor/internal/dev"
	"github.com/brattlof/zeptor/internal/scaffold"
	"github.com/brattlof/zeptor/pkg/plugin"
)

var rootCmd = &cobra.Command{
	Use:     "zt",
	Short:   "Zeptor CLI - Next.js-like framework for Go with eBPF",
	Version: version,
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development server with hot reload",
	Long: `Start the Zeptor development server with:
- Hot module replacement for templ files
- Auto-reload on Go file changes
- eBPF program auto-recompilation`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		nobpf, _ := cmd.Flags().GetBool("no-ebpf")
		configPath, _ := cmd.Flags().GetString("config")

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if port != 3000 {
			cfg.App.Port = port
		}
		if nobpf {
			cfg.EBPF.Enabled = false
		}

		registry := plugin.NewRegistry(slog.Default())
		if len(cfg.Plugins.Enabled) > 0 {
			loader := plugin.NewLoader(registry, cfg.Plugins.Dir, slog.Default())
			pluginConfigs := make(map[string]plugin.PluginOptions)
			for name, opts := range cfg.Plugins.Config {
				pluginConfigs[name] = plugin.PluginOptions(opts)
			}
			if err := loader.LoadFromConfig(context.Background(), cfg.Plugins.Enabled, pluginConfigs); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading plugins: %v\n", err)
				os.Exit(1)
			}
			defer loader.Close()
		}

		if err := dev.RunDev(cfg, registry); err != nil {
			fmt.Fprintf(os.Stderr, "Dev server error: %v\n", err)
			os.Exit(1)
		}
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build for production",
	Long: `Build the Zeptor application:
1. Generate templ components
2. Compile eBPF programs
3. Pre-render SSG pages
4. Build optimized Go binary`,
	Run: func(cmd *cobra.Command, args []string) {
		ssg, _ := cmd.Flags().GetBool("ssg")
		outDir, _ := cmd.Flags().GetString("out")

		fmt.Printf("Building (SSG: %v, out: %s)\n", ssg, outDir)
		fmt.Println("Build not yet implemented - coming in Phase 4")
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start production server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		fmt.Printf("Starting production server on :%d\n", port)
		fmt.Println("Start not yet implemented - run 'go run ./cmd/zeptor' instead")
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [type] [name]",
	Short: "Generate code scaffolds",
	Long: `Generate various code scaffolds:

  zt generate page about        Create app/about/page.templ
  zt generate api users         Create app/api/users/route.go
  zt generate layout admin      Create app/admin/layout.templ
  zt generate component Button  Create components/Button.templ`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		genType := args[0]
		name := args[1]

		switch genType {
		case "page", "p":
			fmt.Printf("Creating page: app/%s/page.templ\n", name)
			fmt.Println("Generate not yet implemented - coming in Phase 5")
		case "api", "a":
			fmt.Printf("Creating API route: app/api/%s/route.go\n", name)
			fmt.Println("Generate not yet implemented - coming in Phase 5")
		case "layout", "l":
			fmt.Printf("Creating layout: app/%s/layout.templ\n", name)
			fmt.Println("Generate not yet implemented - coming in Phase 5")
		case "component", "c":
			fmt.Printf("Creating component: components/%s.templ\n", name)
			fmt.Println("Generate not yet implemented - coming in Phase 5")
		default:
			fmt.Printf("Unknown type: %s\n", genType)
			os.Exit(1)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Zeptor CLI v%s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
	},
}

var createCmd = &cobra.Command{
	Use:   "create [project-name]",
	Short: "Create a new Zeptor project",
	Long: `Create a new Zeptor project from a template.

Examples:
  zt create my-app                    Create a minimal project
  zt create my-api -t api             Create an API-only project
  zt create my-site -t basic -p 8080  Create a basic project on port 8080`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		template, _ := cmd.Flags().GetString("template")
		port, _ := cmd.Flags().GetInt("port")
		skipGit, _ := cmd.Flags().GetBool("skip-git")
		skipTempl, _ := cmd.Flags().GetBool("skip-templ")
		outputDir, _ := cmd.Flags().GetString("output")

		opts := scaffold.Options{
			ProjectName: projectName,
			Template:    template,
			Port:        port,
			SkipGit:     skipGit,
			SkipTempl:   skipTempl,
			OutputDir:   outputDir,
		}

		fmt.Printf("Creating project %s...\n", projectName)

		if err := scaffold.Create(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		targetDir := outputDir
		if targetDir == "" {
			targetDir = projectName
		}

		fmt.Printf("\n✓ Project created in %s/\n\n", targetDir)
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", targetDir)
		fmt.Println("  zt dev")
	},
}

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List all routes discovered from app directory",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		rt, err := router.New(cfg.Routing.AppDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating router: %v\n", err)
			os.Exit(1)
		}

		routes := rt.Routes()

		if jsonOutput {
			output := make([]map[string]interface{}, len(routes))
			for i, r := range routes {
				routeType := "page"
				if r.Type == router.RouteTypeAPI {
					routeType = "api"
				} else if r.Type == router.RouteTypeLayout {
					routeType = "layout"
				}
				output[i] = map[string]interface{}{
					"pattern": r.Pattern,
					"type":    routeType,
					"file":    r.File,
					"dynamic": r.IsDynamic,
					"params":  r.Params,
				}
			}
			data, _ := json.MarshalIndent(map[string]interface{}{"routes": output}, "", "  ")
			fmt.Println(string(data))
			return
		}

		if len(routes) == 0 {
			fmt.Println("No routes found in", cfg.Routing.AppDir)
			return
		}

		fmt.Printf("Found %d route(s) in %s:\n\n", len(routes), cfg.Routing.AppDir)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "METHOD\tPATTERN\tTYPE\tFILE")
		fmt.Fprintln(w, "------\t-------\t----\t----")

		for _, r := range routes {
			method := "GET"
			if r.Type == router.RouteTypeAPI {
				method = "*"
			}

			routeType := "page"
			if r.Type == router.RouteTypeAPI {
				routeType = "api"
			} else if r.Type == router.RouteTypeLayout {
				routeType = "layout"
			}

			dynamic := ""
			if r.IsDynamic {
				dynamic = " [dynamic]"
			}

			fmt.Fprintf(w, "%s\t%s%s\t%s\t%s\n", method, r.Pattern, dynamic, routeType, r.File)
		}
		w.Flush()

		fmt.Printf("\nLayouts: %d\n", len(rt.Layouts()))
		for _, l := range rt.Layouts() {
			fmt.Printf("  %s -> %s\n", l.Pattern, l.File)
		}
	},
}

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `List, inspect, and manage Zeptor plugins.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available plugins",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		registry := plugin.NewRegistry(slog.Default())
		if len(cfg.Plugins.Enabled) > 0 {
			loader := plugin.NewLoader(registry, cfg.Plugins.Dir, slog.Default())
			pluginConfigs := make(map[string]plugin.PluginOptions)
			for name, opts := range cfg.Plugins.Config {
				pluginConfigs[name] = plugin.PluginOptions(opts)
			}
			loader.LoadFromConfig(context.Background(), cfg.Plugins.Enabled, pluginConfigs)
		}

		infos := registry.AllInfo()

		if jsonOutput {
			data, _ := json.MarshalIndent(map[string]interface{}{"plugins": infos}, "", "  ")
			fmt.Println(string(data))
			return
		}

		if len(infos) == 0 {
			fmt.Println("No plugins loaded")
			return
		}

		fmt.Printf("Loaded plugins (%d):\n\n", len(infos))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tHOOKS")
		fmt.Fprintln(w, "----\t-------\t-----")

		for _, info := range infos {
			hooks := ""
			for i, h := range info.Hooks {
				if i > 0 {
					hooks += ", "
				}
				hooks += string(h)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", info.Name, info.Version, hooks)
		}
		w.Flush()
	},
}

var pluginInspectCmd = &cobra.Command{
	Use:   "inspect [plugin-name]",
	Short: "Show detailed plugin information",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		pluginName := args[0]

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		registry := plugin.NewRegistry(slog.Default())
		if len(cfg.Plugins.Enabled) > 0 {
			loader := plugin.NewLoader(registry, cfg.Plugins.Dir, slog.Default())
			pluginConfigs := make(map[string]plugin.PluginOptions)
			for name, opts := range cfg.Plugins.Config {
				pluginConfigs[name] = plugin.PluginOptions(opts)
			}
			loader.LoadFromConfig(context.Background(), cfg.Plugins.Enabled, pluginConfigs)
		}

		info, ok := registry.Info(pluginName)
		if !ok {
			fmt.Fprintf(os.Stderr, "Plugin %q not found\n", pluginName)
			os.Exit(1)
		}

		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Description: %s\n", info.Description)
		fmt.Printf("Enabled: %v\n", info.Enabled)

		if len(info.Hooks) > 0 {
			fmt.Println("\nHooks:")
			for _, h := range info.Hooks {
				fmt.Printf("  - %s\n", h)
			}
		}

		if len(info.Config) > 0 {
			fmt.Println("\nConfiguration:")
			for k, v := range info.Config {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	},
}

func init() {
	devCmd.Flags().IntP("port", "p", 3000, "Port to run dev server on")
	devCmd.Flags().Bool("no-ebpf", false, "Disable eBPF acceleration")
	devCmd.Flags().StringP("config", "c", "", "Path to config file")

	buildCmd.Flags().Bool("ssg", false, "Enable static site generation")
	buildCmd.Flags().StringP("out", "o", "./dist", "Output directory")
	buildCmd.Flags().StringP("config", "c", "", "Path to config file")

	startCmd.Flags().IntP("port", "p", 3000, "Port to run server on")
	startCmd.Flags().StringP("config", "c", "", "Path to config file")

	routesCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	routesCmd.Flags().StringP("config", "c", "", "Path to config file")

	createCmd.Flags().StringP("template", "t", "minimal", "Project template (minimal, basic, api)")
	createCmd.Flags().IntP("port", "p", 3000, "Default port for the application")
	createCmd.Flags().Bool("skip-git", false, "Skip git initialization")
	createCmd.Flags().Bool("skip-templ", false, "Skip templ generate")
	createCmd.Flags().StringP("output", "o", "", "Output directory (default: project name)")

	pluginListCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	pluginListCmd.Flags().StringP("config", "c", "", "Path to config file")

	pluginInspectCmd.Flags().StringP("config", "c", "", "Path to config file")

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInspectCmd)

	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(routesCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(pluginCmd)
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
