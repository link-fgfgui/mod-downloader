package cliapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"mod-downloader/appcore"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"
	structs "mod-downloader/structs/minecraft"
)

type runner struct {
	stdout io.Writer
	stderr io.Writer
}

func New(stdout, stderr io.Writer) *cli.App {
	r := runner{stdout: stdout, stderr: stderr}
	app := &cli.App{
		Name:  "mod-downloader-cli",
		Usage: "Search, inspect, and install Minecraft mods without the desktop UI",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "write structured JSON output"},
			&cli.StringFlag{Name: "minecraft-dir", Usage: "override the Minecraft root directory"},
			&cli.StringFlag{Name: "curseforge-api-key", Usage: "override the CurseForge API key"},
			&cli.StringFlag{Name: "modrinth-api-key", Usage: "override the Modrinth API key"},
		},
		Commands: []*cli.Command{
			r.configCommand(),
			r.versionsCommand(),
			r.searchCommand(),
			r.installCommand(),
			r.modsCommand(),
		},
	}
	app.Writer = stdout
	app.ErrWriter = stderr
	return app
}

func (r runner) configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Show or update effective configuration",
		Flags: []cli.Flag{
			jsonFlag(),
			&cli.StringFlag{Name: "set-minecraft-dir", Usage: "persist a Minecraft root directory"},
			&cli.StringFlag{Name: "theme", Usage: "persist theme preference: dark, light, or system"},
			&cli.StringFlag{Name: "set-curseforge-api-key", Usage: "persist a CurseForge API key"},
			&cli.StringFlag{Name: "set-modrinth-api-key", Usage: "persist a Modrinth API key"},
			&cli.BoolFlag{Name: "clear-curseforge-api-key", Usage: "clear the persisted CurseForge API key"},
			&cli.BoolFlag{Name: "clear-modrinth-api-key", Usage: "clear the persisted Modrinth API key"},
		},
		Action: func(c *cli.Context) error {
			return r.withService(c, func(ctx context.Context, svc *appcore.Service) error {
				if dir := strings.TrimSpace(c.String("set-minecraft-dir")); dir != "" {
					svc.SaveMinecraftDirPreference(dir)
				}
				if theme := strings.TrimSpace(c.String("theme")); theme != "" {
					svc.SaveTheme(theme)
				}
				keyReq := appcore.SaveApiKeysRequest{
					CurseforgeApiKey: appcore.APIKeyKeepSentinel,
					ModrinthApiKey:   appcore.APIKeyKeepSentinel,
				}
				if c.Bool("clear-curseforge-api-key") {
					keyReq.CurseforgeApiKey = ""
				}
				if c.Bool("clear-modrinth-api-key") {
					keyReq.ModrinthApiKey = ""
				}
				if c.IsSet("set-curseforge-api-key") {
					keyReq.CurseforgeApiKey = c.String("set-curseforge-api-key")
				}
				if c.IsSet("set-modrinth-api-key") {
					keyReq.ModrinthApiKey = c.String("set-modrinth-api-key")
				}
				if keyReq.CurseforgeApiKey != appcore.APIKeyKeepSentinel || keyReq.ModrinthApiKey != appcore.APIKeyKeepSentinel {
					svc.SaveApiKeys(keyReq)
				}
				return r.write(c, svc.GetSettings(), writeSettings)
			})
		},
	}
}

func (r runner) versionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "versions",
		Usage: "List supported Minecraft instances",
		Flags: []cli.Flag{jsonFlag()},
		Action: func(c *cli.Context) error {
			return r.withService(c, func(ctx context.Context, svc *appcore.Service) error {
				versions := svc.GetVersions()
				return r.write(c, versions, writeVersions)
			})
		},
	}
}

func (r runner) searchCommand() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search Modrinth and configured CurseForge metadata",
		ArgsUsage: "[query]",
		Flags: []cli.Flag{
			jsonFlag(),
			&cli.StringFlag{Name: "query", Usage: "search query; defaults to the first positional argument"},
			&cli.StringFlag{Name: "version", Aliases: []string{"minecraft-version"}, Usage: "Minecraft version filter"},
			&cli.StringFlag{Name: "loader", Usage: "mod loader filter: fabric, forge, or neoforge"},
			&cli.IntFlag{Name: "limit", Value: 10, Usage: "maximum result count per provider"},
			&cli.IntFlag{Name: "offset", Usage: "result offset"},
		},
		Action: func(c *cli.Context) error {
			return r.withService(c, func(ctx context.Context, svc *appcore.Service) error {
				query := strings.TrimSpace(c.String("query"))
				if query == "" {
					query = strings.TrimSpace(c.Args().First())
				}
				update := svc.SearchModsCollect(appstructs.SearchModsRequest{
					Query:     query,
					Version:   c.String("version"),
					ModLoader: c.String("loader"),
					Limit:     c.Int("limit"),
					Offset:    c.Int("offset"),
				})
				return r.write(c, update.Results, writeProjects)
			})
		},
	}
}

func (r runner) installCommand() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install a project into a target instance",
		Flags: []cli.Flag{
			jsonFlag(),
			&cli.StringFlag{Name: "project", Usage: "project ID, slug, or platform-prefixed reference such as modrinth:sodium", Required: true},
			&cli.StringFlag{Name: "platform", Usage: "project platform: modrinth or curseforge"},
			&cli.StringFlag{Name: "instance", Usage: "target version/instance key from the versions command", Required: true},
			&cli.StringFlag{Name: "version-id", Usage: "explicit platform version ID to install"},
			&cli.DurationFlag{Name: "timeout", Usage: "optional install timeout, for example 2m"},
		},
		Action: func(c *cli.Context) error {
			return r.withService(c, func(ctx context.Context, svc *appcore.Service) error {
				if timeout := c.Duration("timeout"); timeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()
				}
				selected, err := svc.SelectVersion(c.String("instance"))
				if err != nil {
					return err
				}
				project, err := resolveProject(c, svc)
				if err != nil {
					return err
				}
				result := svc.InstallModAndWait(ctx, appstructs.ModDownloadRequest{
					ProjectID:        project.ID,
					Result:           project,
					MinecraftVersion: selected.MinecraftVersion,
					ModLoader:        selected.ModLoader,
					VersionID:        c.String("version-id"),
				})
				if len(result.Errors) > 0 {
					return cli.Exit(result.Errors[0].Reason, 1)
				}
				if result.Result.Skipped {
					return cli.Exit(result.Result.Reason, 1)
				}
				return r.write(c, result, writeInstallResult)
			})
		},
	}
}

func (r runner) modsCommand() *cli.Command {
	return &cli.Command{
		Name:  "mods",
		Usage: "List installed local mods for an instance",
		Flags: []cli.Flag{
			jsonFlag(),
			&cli.StringFlag{Name: "instance", Usage: "version/instance key from the versions command"},
		},
		Action: func(c *cli.Context) error {
			return r.withService(c, func(ctx context.Context, svc *appcore.Service) error {
				svc.GetVersions()
				mods, err := svc.LocalMods(c.String("instance"))
				if err != nil {
					return err
				}
				return r.write(c, mods, writeMods)
			})
		},
	}
}

func (r runner) withService(c *cli.Context, fn func(context.Context, *appcore.Service) error) error {
	overrides := appcore.ConfigOverrides{}
	if c.IsSet("minecraft-dir") {
		overrides.MinecraftDir = c.String("minecraft-dir")
		overrides.HasMinecraftDir = true
	}
	if c.IsSet("curseforge-api-key") {
		overrides.CurseForgeAPIKey = c.String("curseforge-api-key")
		overrides.HasCurseForgeAPIKey = true
	}
	if c.IsSet("modrinth-api-key") {
		overrides.ModrinthAPIKey = c.String("modrinth-api-key")
		overrides.HasModrinthAPIKey = true
	}
	svc := appcore.New(appcore.Options{ConfigOverrides: overrides})
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := svc.Startup(ctx); err != nil {
		return err
	}
	defer svc.Close()
	return fn(ctx, svc)
}

func jsonFlag() cli.Flag {
	return &cli.BoolFlag{Name: "json", Usage: "write structured JSON output"}
}

func (r runner) write(c *cli.Context, value any, human func(io.Writer, any) error) error {
	if c.Bool("json") {
		encoder := json.NewEncoder(r.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	return human(r.stdout, value)
}

func resolveProject(c *cli.Context, svc *appcore.Service) (models.ModProject, error) {
	ref := strings.TrimSpace(c.String("project"))
	if ref == "" {
		return models.ModProject{}, errors.New("project is required")
	}
	platform := strings.ToLower(strings.TrimSpace(c.String("platform")))
	parsedPlatform, projectID := models.ParseProjectKey(ref)
	if platform == "" {
		platform = parsedPlatform
	}
	if parsedPlatform != "" {
		ref = projectID
	}
	if platform == "" {
		return models.ModProject{}, errors.New("platform is required when project is not prefixed")
	}
	selected := svc.GetSelectedVersion()
	project, ok := svc.LookupProject(platform, ref, selected.MinecraftVersion, selected.ModLoader)
	if ok {
		return project, nil
	}
	return models.ModProject{
		ID:        models.ProjectKey(platform, ref),
		Platform:  platform,
		ProjectID: ref,
		Slug:      ref,
		Title:     ref,
	}, nil
}

func writeSettings(w io.Writer, value any) error {
	settings := value.(appcore.SettingsView)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "THEME\t%s\n", settings.Theme)
	fmt.Fprintf(tw, "MINECRAFT_DIR\t%s\n", settings.MinecraftDir)
	fmt.Fprintf(tw, "CURSEFORGE_KEY\t%s\n", keyState(settings.HasCurseforgeKey, settings.CurseforgeKeyMask))
	fmt.Fprintf(tw, "MODRINTH_KEY\t%s\n", keyState(settings.HasModrinthKey, settings.ModrinthKeyMask))
	return tw.Flush()
}

func writeVersions(w io.Writer, value any) error {
	versions := value.([]structs.VersionInfo)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "KEY\tNAME\tMINECRAFT\tLOADER")
	for _, version := range versions {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", version.ID, version.Name, version.MinecraftVersion, version.ModLoader)
	}
	return tw.Flush()
}

func writeProjects(w io.Writer, value any) error {
	projects := value.([]models.ModProject)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPLATFORM\tTITLE\tSLUG")
	for _, project := range projects {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", project.ID, project.Platform, project.Title, project.Slug)
	}
	return tw.Flush()
}

func writeInstallResult(w io.Writer, value any) error {
	result := value.(appcore.InstallWaitResult)
	if result.Result.Queued {
		fmt.Fprintf(w, "installed %s (%s)\n", result.Result.FileName, result.Result.VersionID)
		return nil
	}
	if result.Result.Skipped {
		fmt.Fprintf(w, "skipped: %s\n", result.Result.Reason)
		return nil
	}
	fmt.Fprintln(w, "no download queued")
	return nil
}

func writeMods(w io.Writer, value any) error {
	mods := value.([]structs.ModInfo)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tVERSION\tFILE\tENABLED")
	for _, mod := range mods {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\n", mod.ID, mod.Name, mod.Version, mod.FileName, mod.Enabled)
	}
	return tw.Flush()
}

func keyState(ok bool, mask string) string {
	if !ok {
		return "not set"
	}
	if mask == "" {
		return "set"
	}
	return mask
}
