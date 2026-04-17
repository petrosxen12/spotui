package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/petrosxen/spotui/internal/auth"
	"github.com/petrosxen/spotui/internal/config"
	spotifyapi "github.com/petrosxen/spotui/internal/spotify"
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
		Short:         "CLI Spotify controller for Phase 1",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newMeCmd())
	rootCmd.AddCommand(newDevicesCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newPlayCmd())
	rootCmd.AddCommand(newPauseCmd())
	rootCmd.AddCommand(newResumeCmd())
	rootCmd.AddCommand(newNextCmd())
	rootCmd.AddCommand(newPrevCmd())

	return rootCmd
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				me, err := client.Me(cmd.Context())
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				devices, err := client.Devices(cmd.Context())
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				return runUse(cmd.Context(), cfg, client, strings.Join(args, " "))
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				return runSearch(cmd.Context(), cfg, client, strings.Join(args, " "))
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				return runPlay(cmd.Context(), cfg, client, "track", args[0])
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				return runPlay(cmd.Context(), cfg, client, "playlist", args[0])
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				deviceID, err := client.ResolvePlaybackDevice(cmd.Context(), cfg.PreferredDeviceID)
				if err != nil {
					return err
				}
				return client.Pause(cmd.Context(), deviceID)
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				deviceID, err := client.ResolvePlaybackDevice(cmd.Context(), cfg.PreferredDeviceID)
				if err != nil {
					return err
				}
				return client.Resume(cmd.Context(), deviceID)
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				deviceID, err := client.ResolvePlaybackDevice(cmd.Context(), cfg.PreferredDeviceID)
				if err != nil {
					return err
				}
				return client.Next(cmd.Context(), deviceID)
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
			return withClient(cmd.Context(), cfg, func(client *spotifyapi.Client) error {
				deviceID, err := client.ResolvePlaybackDevice(cmd.Context(), cfg.PreferredDeviceID)
				if err != nil {
					return err
				}
				return client.Previous(cmd.Context(), deviceID)
			})
		},
	}
}

func runUse(ctx context.Context, cfg *config.Config, client *spotifyapi.Client, needle string) error {
	devices, err := client.Devices(ctx)
	if err != nil {
		return err
	}

	var matches []spotifyapi.Device
	for _, device := range devices {
		if strings.Contains(strings.ToLower(device.Name), strings.ToLower(needle)) {
			matches = append(matches, device)
		}
	}

	switch len(matches) {
	case 0:
		return fmt.Errorf("no device matched %q", needle)
	case 1:
		cfg.PreferredDeviceID = matches[0].ID
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Preferred device set to %s (%s)\n", matches[0].Name, matches[0].ID)
		return nil
	default:
		var names []string
		for _, match := range matches {
			names = append(names, fmt.Sprintf("%s (%s)", match.Name, match.ID))
		}
		return fmt.Errorf("multiple devices matched %q: %s", needle, strings.Join(names, ", "))
	}
}

func runSearch(ctx context.Context, cfg *config.Config, client *spotifyapi.Client, query string) error {
	results, err := client.Search(ctx, query)
	if err != nil {
		return err
	}

	cfg.LastSearch = config.LastSearch{
		Query:      query,
		Tracks:     results.Tracks,
		Playlists:  results.Playlists,
		SearchedAt: time.Now().UTC(),
	}
	if err := config.Save(cfg); err != nil {
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

func runPlay(ctx context.Context, cfg *config.Config, client *spotifyapi.Client, kind string, ref string) error {
	deviceID, err := client.ResolvePlaybackDevice(ctx, cfg.PreferredDeviceID)
	if err != nil {
		return err
	}

	switch kind {
	case "track":
		uri, err := resolveTrackRef(cfg, ref)
		if err != nil {
			return err
		}
		return client.PlayTrack(ctx, deviceID, uri)
	case "playlist":
		uri, err := resolvePlaylistRef(cfg, ref)
		if err != nil {
			return err
		}
		return client.PlayPlaylist(ctx, deviceID, uri)
	default:
		return fmt.Errorf("unknown play target %q; expected track or playlist", kind)
	}
}

func resolveTrackRef(cfg *config.Config, ref string) (string, error) {
	if idx, err := strconv.Atoi(ref); err == nil {
		if idx <= 0 || idx > len(cfg.LastSearch.Tracks) {
			return "", fmt.Errorf("track index %d is out of range for the last search", idx)
		}
		return cfg.LastSearch.Tracks[idx-1].URI, nil
	}
	return normalizePlayableRef(ref, "track"), nil
}

func resolvePlaylistRef(cfg *config.Config, ref string) (string, error) {
	if idx, err := strconv.Atoi(ref); err == nil {
		if idx <= 0 || idx > len(cfg.LastSearch.Playlists) {
			return "", fmt.Errorf("playlist index %d is out of range for the last search", idx)
		}
		return cfg.LastSearch.Playlists[idx-1].URI, nil
	}
	return normalizePlayableRef(ref, "playlist"), nil
}

func normalizePlayableRef(ref string, kind string) string {
	if strings.HasPrefix(ref, "spotify:") {
		return ref
	}
	return fmt.Sprintf("spotify:%s:%s", kind, ref)
}

func withClient(ctx context.Context, cfg *config.Config, fn func(client *spotifyapi.Client) error) error {
	manager, err := auth.NewManager(cfg)
	if err != nil {
		return err
	}
	client := spotifyapi.NewClient(cfg, manager)
	return fn(client)
}
