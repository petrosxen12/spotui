package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/auth"
	"github.com/petrosxen/spotui/internal/config"
	"github.com/petrosxen/spotui/internal/ui"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "spotui: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "spotui",
		Short:         "Spotify controller with CLI and TUI modes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newMeCmd())
	rootCmd.AddCommand(newDevicesCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newPlayCmd())
	rootCmd.AddCommand(newPauseCmd())
	rootCmd.AddCommand(newResumeCmd())
	rootCmd.AddCommand(newNextCmd())
	rootCmd.AddCommand(newPrevCmd())
	rootCmd.AddCommand(newLocalCmd())

	return rootCmd
}

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the Bubble Tea terminal UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			service, err := app.NewPlayerService(cfg)
			if err != nil {
				return err
			}
			return ui.Run(service)
		},
	}
}

func newLoginCmd() *cobra.Command {
	var clientID string
	var redirectURI string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Spotify using Authorization Code with PKCE",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if clientID == "" {
				clientID = cfg.ClientID
			}
			if redirectURI == "" {
				redirectURI = cfg.RedirectURI
			}
			if clientID == "" {
				return errors.New("missing Spotify client ID; set SPOTUI_CLIENT_ID, config.json, or pass --client-id")
			}
			if redirectURI == "" {
				redirectURI = config.DefaultRedirectURI
			}

			cfg.ClientID = clientID
			cfg.RedirectURI = redirectURI
			if err := config.Save(cfg); err != nil {
				return err
			}

			manager, err := auth.NewManager(cfg)
			if err != nil {
				return err
			}
			token, err := manager.Login(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("Login complete. Access token expires at %s.\n", token.ExpiresAt.Format(time.RFC3339))
			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "Spotify app client ID")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "", "OAuth redirect URI")
	return cmd
}

func newMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Print the current Spotify user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				me, err := service.CurrentUser(cmd.Context())
				if err != nil {
					return err
				}
				if me.DisplayName != "" {
					fmt.Printf("%s (%s)\n", me.ID, me.DisplayName)
				} else {
					fmt.Printf("%s\n", me.ID)
				}
				return nil
			})
		},
	}
}

func newDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List available Spotify playback devices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				devices, err := service.ListDevices(cmd.Context())
				if err != nil {
					return err
				}
				if len(devices) == 0 {
					fmt.Println("No devices available.")
					return nil
				}
				for _, device := range devices {
					active := "inactive"
					if device.IsActive {
						active = "active"
					}
					fmt.Printf("%s\t%s\t%s\t%s\n", device.ID, device.Name, device.Type, active)
				}
				return nil
			})
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <substring>",
		Short: "Set the preferred playback device by substring match",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return runUse(cmd.Context(), service, strings.Join(args, " "))
			})
		},
	}
}

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search for tracks and playlists",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return runSearch(cmd.Context(), service, strings.Join(args, " "))
			})
		},
	}
}

func newPlayCmd() *cobra.Command {
	playCmd := &cobra.Command{
		Use:   "play",
		Short: "Start playback on the selected device",
	}

	playCmd.AddCommand(&cobra.Command{
		Use:   "track <spotify-track-id|spotify-track-uri|last-search-track-index>",
		Short: "Play a track",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return runPlayTrack(cmd.Context(), service, args[0])
			})
		},
	})

	playCmd.AddCommand(&cobra.Command{
		Use:   "playlist <spotify-playlist-id|spotify-playlist-uri|last-search-playlist-index>",
		Short: "Play a playlist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return runPlayPlaylist(cmd.Context(), service, args[0])
			})
		},
	})

	return playCmd
}

func newPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause playback",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return service.Pause(cmd.Context())
			})
		},
	}
}

func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume playback",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return service.Resume(cmd.Context())
			})
		},
	}
}

func newNextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Skip to the next item",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return service.Next(cmd.Context())
			})
		},
	}
}

func newPrevCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prev",
		Short: "Return to the previous item",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				return service.Prev(cmd.Context())
			})
		},
	}
}

func newLocalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Manage the local spotifyd playback daemon",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show local-player status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				status, err := service.LocalPlayerStatus(cmd.Context())
				if err != nil {
					return err
				}
				fmt.Printf("Enabled:\t%t\n", status.Enabled)
				fmt.Printf("Binary available:\t%t\n", status.Binary.Available)
				fmt.Printf("State:\t%s\n", status.Process.State)
				if status.ConfiguredName != "" {
					fmt.Printf("Configured device:\t%s\n", status.ConfiguredName)
				}
				if status.Device.Name != "" {
					fmt.Printf("Spotify device:\t%s (%s)\n", status.Device.Name, status.Device.ID)
				}
				if status.LogPath != "" {
					fmt.Printf("Log file:\t%s\n", status.LogPath)
				}
				if status.Message.Text != "" {
					fmt.Printf("Message:\t%s\n", status.Message.Text)
				}
				return nil
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start the managed local spotifyd player",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				if err := service.StartLocalPlayer(cmd.Context()); err != nil {
					return err
				}
				fmt.Println("Local spotifyd player started and selected.")
				return nil
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the managed local spotifyd player",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				if err := service.StopLocalPlayer(cmd.Context()); err != nil {
					return err
				}
				fmt.Println("Local spotifyd player stopped.")
				return nil
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "use",
		Short: "Start and select the managed local spotifyd player",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				if err := service.UseLocalPlayer(cmd.Context()); err != nil {
					return err
				}
				fmt.Println("Local spotifyd player selected.")
				return nil
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Stop the current spotifyd process and clear managed runtime metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return withService(cfg, func(service app.PlayerService) error {
				if err := service.ResetLocalPlayer(cmd.Context()); err != nil {
					return err
				}
				fmt.Println("Local spotifyd runtime was reset.")
				return nil
			})
		},
	})

	return cmd
}

func runUse(ctx context.Context, service app.PlayerService, needle string) error {
	device, err := service.SetDeviceByName(ctx, needle)
	if err != nil {
		return err
	}
	fmt.Printf("Preferred device set to %s (%s)\n", device.Name, device.ID)
	return nil
}

func runSearch(ctx context.Context, service app.PlayerService, query string) error {
	results, err := service.Search(ctx, query)
	if err != nil {
		return err
	}

	fmt.Printf("Tracks for %q:\n", query)
	if len(results.Tracks) == 0 {
		fmt.Println("  none")
	} else {
		for i, track := range results.Tracks {
			fmt.Printf("  [%d] %s | %s | %s\n", i+1, track.URI, track.ID, track.Name)
		}
	}

	fmt.Printf("Playlists for %q:\n", query)
	if len(results.Playlists) == 0 {
		fmt.Println("  none")
	} else {
		for i, playlist := range results.Playlists {
			fmt.Printf("  [%d] %s | %s | %s\n", i+1, playlist.URI, playlist.ID, playlist.Name)
		}
	}

	return nil
}

func runPlayTrack(ctx context.Context, service app.PlayerService, ref string) error {
	return service.PlayTrack(ctx, ref)
}

func runPlayPlaylist(ctx context.Context, service app.PlayerService, ref string) error {
	return service.PlayPlaylist(ctx, ref)
}

func withService(cfg *config.Config, fn func(service app.PlayerService) error) error {
	service, err := app.NewPlayerService(cfg)
	if err != nil {
		return err
	}
	return fn(service)
}
