package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/cli/cliflag"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/codersdk"
)

const (
	goosWindows = "windows"
	goosDarwin  = "darwin"
)

func init() {
	// Hide output from the browser library,
	// otherwise we can get really verbose and non-actionable messages
	// when in SSH or another type of headless session
	// NOTE: This needs to be in `init` to prevent data races
	// (multiple threads trying to set the global browser.Std* variables)
	browser.Stderr = io.Discard
	browser.Stdout = io.Discard
}

func login() *cobra.Command {
	var (
		email    string
		username string
		password string
	)
	cmd := &cobra.Command{
		Use:   "login <url>",
		Short: "Authenticate with a Coder deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawURL := args[0]

			if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
				scheme := "https"
				if strings.HasPrefix(rawURL, "localhost") {
					scheme = "http"
				}
				rawURL = fmt.Sprintf("%s://%s", scheme, rawURL)
			}
			serverURL, err := url.Parse(rawURL)
			if err != nil {
				return xerrors.Errorf("parse raw url %q: %w", rawURL, err)
			}
			// Default to HTTPs. Enables simple URLs like: master.cdr.dev
			if serverURL.Scheme == "" {
				serverURL.Scheme = "https"
			}

			client := codersdk.New(serverURL)

			// Try to check the version of the server prior to logging in.
			// It may be useful to warn the user if they are trying to login
			// on a very old client.
			err = checkVersions(cmd, client)
			if err != nil {
				return xerrors.Errorf("check versions: %w", err)
			}

			hasInitialUser, err := client.HasFirstUser(cmd.Context())
			if err != nil {
				return xerrors.Errorf("has initial user: %w", err)
			}
			if !hasInitialUser {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), caret+"Your Coder deployment hasn't been set up!\n")

				if username == "" {
					if !isTTY(cmd) {
						return xerrors.New("the initial user cannot be created in non-interactive mode. use the API")
					}
					_, err := cliui.Prompt(cmd, cliui.PromptOptions{
						Text:      "Would you like to create the first user?",
						Default:   cliui.ConfirmYes,
						IsConfirm: true,
					})
					if errors.Is(err, cliui.Canceled) {
						return nil
					}
					if err != nil {
						return err
					}
					currentUser, err := user.Current()
					if err != nil {
						return xerrors.Errorf("get current user: %w", err)
					}
					username, err = cliui.Prompt(cmd, cliui.PromptOptions{
						Text:    "What " + cliui.Styles.Field.Render("username") + " would you like?",
						Default: currentUser.Username,
					})
					if errors.Is(err, cliui.Canceled) {
						return nil
					}
					if err != nil {
						return xerrors.Errorf("pick username prompt: %w", err)
					}
				}

				if email == "" {
					email, err = cliui.Prompt(cmd, cliui.PromptOptions{
						Text: "What's your " + cliui.Styles.Field.Render("email") + "?",
						Validate: func(s string) error {
							err := validator.New().Var(s, "email")
							if err != nil {
								return xerrors.New("That's not a valid email address!")
							}
							return err
						},
					})
					if err != nil {
						return xerrors.Errorf("specify email prompt: %w", err)
					}
				}

				if password == "" {
					var matching bool

					for !matching {
						password, err = cliui.Prompt(cmd, cliui.PromptOptions{
							Text:     "Enter a " + cliui.Styles.Field.Render("password") + ":",
							Secret:   true,
							Validate: cliui.ValidateNotEmpty,
						})
						if err != nil {
							return xerrors.Errorf("specify password prompt: %w", err)
						}
						confirm, err := cliui.Prompt(cmd, cliui.PromptOptions{
							Text:   "Confirm " + cliui.Styles.Field.Render("password") + ":",
							Secret: true,
						})
						if err != nil {
							return xerrors.Errorf("confirm password prompt: %w", err)
						}

						matching = confirm == password
						if !matching {
							_, _ = fmt.Fprintln(cmd.OutOrStdout(), cliui.Styles.Error.Render("Passwords do not match"))
						}
					}
				}

				_, err = client.CreateFirstUser(cmd.Context(), codersdk.CreateFirstUserRequest{
					Email:            email,
					Username:         username,
					OrganizationName: username,
					Password:         password,
				})
				if err != nil {
					return xerrors.Errorf("create initial user: %w", err)
				}
				resp, err := client.LoginWithPassword(cmd.Context(), codersdk.LoginWithPasswordRequest{
					Email:    email,
					Password: password,
				})
				if err != nil {
					return xerrors.Errorf("login with password: %w", err)
				}

				sessionToken := resp.SessionToken
				config := createConfig(cmd)
				err = config.Session().Write(sessionToken)
				if err != nil {
					return xerrors.Errorf("write session token: %w", err)
				}
				err = config.URL().Write(serverURL.String())
				if err != nil {
					return xerrors.Errorf("write server url: %w", err)
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					cliui.Styles.Paragraph.Render(fmt.Sprintf("Welcome to Coder, %s! You're authenticated.", cliui.Styles.Keyword.Render(username)))+"\n")

				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					cliui.Styles.Paragraph.Render("Get started by creating a template: "+cliui.Styles.Code.Render("coder templates init"))+"\n")
				return nil
			}

			sessionToken, _ := cmd.Flags().GetString(varToken)
			if sessionToken == "" {
				authURL := *serverURL
				// Don't use filepath.Join, we don't want to use the os separator
				// for a url.
				authURL.Path = path.Join(serverURL.Path, "/cli-auth")
				if err := openURL(cmd, authURL.String()); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open the following in your browser:\n\n\t%s\n\n", authURL.String())
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Your browser has been opened to visit:\n\n\t%s\n\n", authURL.String())
				}

				sessionToken, err = cliui.Prompt(cmd, cliui.PromptOptions{
					Text:   "Paste your token here:",
					Secret: true,
					Validate: func(token string) error {
						client.SessionToken = token
						_, err := client.User(cmd.Context(), codersdk.Me)
						if err != nil {
							return xerrors.New("That's not a valid token!")
						}
						return err
					},
				})
				if err != nil {
					return xerrors.Errorf("paste token prompt: %w", err)
				}
			}

			// Login to get user data - verify it is OK before persisting
			client.SessionToken = sessionToken
			resp, err := client.User(cmd.Context(), codersdk.Me)
			if err != nil {
				return xerrors.Errorf("get user: %w", err)
			}

			config := createConfig(cmd)
			err = config.Session().Write(sessionToken)
			if err != nil {
				return xerrors.Errorf("write session token: %w", err)
			}
			err = config.URL().Write(serverURL.String())
			if err != nil {
				return xerrors.Errorf("write server url: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), caret+"Welcome to Coder, %s! You're authenticated.\n", cliui.Styles.Keyword.Render(resp.Username))
			return nil
		},
	}
	cliflag.StringVarP(cmd.Flags(), &email, "email", "e", "CODER_EMAIL", "", "Specifies an email address to authenticate with.")
	cliflag.StringVarP(cmd.Flags(), &username, "username", "u", "CODER_USERNAME", "", "Specifies a username to authenticate with.")
	cliflag.StringVarP(cmd.Flags(), &password, "password", "p", "CODER_PASSWORD", "", "Specifies a password to authenticate with.")
	return cmd
}

// isWSL determines if coder-cli is running within Windows Subsystem for Linux
func isWSL() (bool, error) {
	if runtime.GOOS == goosDarwin || runtime.GOOS == goosWindows {
		return false, nil
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false, xerrors.Errorf("read /proc/version: %w", err)
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft"), nil
}

// openURL opens the provided URL via user's default browser
func openURL(cmd *cobra.Command, urlToOpen string) error {
	noOpen, err := cmd.Flags().GetBool(varNoOpen)
	if err != nil {
		panic(err)
	}
	if noOpen {
		return xerrors.New("opening is blocked")
	}
	wsl, err := isWSL()
	if err != nil {
		return xerrors.Errorf("test running Windows Subsystem for Linux: %w", err)
	}

	if wsl {
		// #nosec
		return exec.Command("cmd.exe", "/c", "start", strings.ReplaceAll(urlToOpen, "&", "^&")).Start()
	}

	return browser.OpenURL(urlToOpen)
}
