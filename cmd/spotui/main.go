package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/petrosxen/spotui/internal/auth"
	"github.com/petrosxen/spotui/internal/config"
	spotifyapi "github.com/petrosxen/spotui/internal/spotify"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "spotui: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "login":
		return runLogin(ctx, cfg, args[1:])
	case "me":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			me, err := client.Me(ctx)
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
	case "devices":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			devices, err := client.Devices(ctx)
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
	case "use":
		if len(args) < 2 {
			return errors.New("usage: spotui use <substring>")
		}
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			return runUse(ctx, cfg, client, strings.Join(args[1:], " "))
		})
	case "search":
		if len(args) < 2 {
			return errors.New("usage: spotui search <query>")
		}
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			return runSearch(ctx, cfg, client, strings.Join(args[1:], " "))
		})
	case "play":
		if len(args) != 3 {
			return errors.New("usage: spotui play track|playlist <spotify-id|spotify-uri|last-search-index>")
		}
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			return runPlay(ctx, cfg, client, args[1], args[2])
		})
	case "pause":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			deviceID, err := client.ResolvePlaybackDevice(ctx, cfg.PreferredDeviceID)
			if err != nil {
				return err
			}
			return client.Pause(ctx, deviceID)
		})
	case "resume":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			deviceID, err := client.ResolvePlaybackDevice(ctx, cfg.PreferredDeviceID)
			if err != nil {
				return err
			}
			return client.Resume(ctx, deviceID)
		})
	case "next":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			deviceID, err := client.ResolvePlaybackDevice(ctx, cfg.PreferredDeviceID)
			if err != nil {
				return err
			}
			return client.Next(ctx, deviceID)
		})
	case "prev":
		return withClient(ctx, cfg, func(client *spotifyapi.Client) error {
			deviceID, err := client.ResolvePlaybackDevice(ctx, cfg.PreferredDeviceID)
			if err != nil {
				return err
			}
			return client.Previous(ctx, deviceID)
		})
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runLogin(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	clientID := fs.String("client-id", cfg.ClientID, "Spotify app client ID")
	redirectURI := fs.String("redirect-uri", cfg.RedirectURI, "OAuth redirect URI")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *clientID == "" {
		return errors.New("missing Spotify client ID; set SPOTUI_CLIENT_ID, config.json, or pass --client-id")
	}
	if *redirectURI == "" {
		*redirectURI = config.DefaultRedirectURI
	}

	cfg.ClientID = *clientID
	cfg.RedirectURI = *redirectURI
	if err := config.Save(cfg); err != nil {
		return err
	}

	manager, err := auth.NewManager(cfg)
	if err != nil {
		return err
	}
	token, err := manager.Login(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Login complete. Access token expires at %s.\n", token.ExpiresAt.Format(time.RFC3339))
	return nil
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

func printUsage() {
	fmt.Println(`spotui commands:
  spotui login [--client-id <id>] [--redirect-uri <uri>]
  spotui me
  spotui devices
  spotui use <substring>
  spotui search "<query>"
  spotui play track <spotify-track-id|spotify-track-uri|last-search-track-index>
  spotui play playlist <spotify-playlist-id|spotify-playlist-uri|last-search-playlist-index>
  spotui next
  spotui prev
  spotui pause
  spotui resume`)
}
