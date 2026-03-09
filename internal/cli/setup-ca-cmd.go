package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/nguyennghia/saola-proxy/internal/proxy"
	"github.com/spf13/cobra"
)

var setupCACmd = &cobra.Command{
	Use:   "setup-ca",
	Short: "Generate and install CA certificate for HTTPS proxy",
	Long: `Generates a local CA certificate for Saola's MITM proxy and installs it
into the system trust store.

This is required once before using "saola proxy".

On macOS: installs to System Keychain (requires sudo password).
On Linux: installs to /usr/local/share/ca-certificates/.

The CA cert/key are stored at ~/.saola/ca.crt and ~/.saola/ca.key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if CA already exists.
		if proxy.CAExists() {
			fmt.Println("CA certificate already exists at ~/.saola/ca.crt")
			fmt.Println("To regenerate, delete ~/.saola/ca.crt and ~/.saola/ca.key first.")
			return nil
		}

		// Generate CA cert/key.
		fmt.Println("Generating Saola Proxy CA certificate...")
		certPath, _, err := proxy.GenerateCA()
		if err != nil {
			return fmt.Errorf("generate CA: %w", err)
		}
		fmt.Printf("CA certificate saved to: %s\n", certPath)

		// Install to system trust store.
		fmt.Println("\nInstalling CA to system trust store...")
		if err := installCA(certPath); err != nil {
			fmt.Fprintf(os.Stderr, "Auto-install failed: %v\n", err)
			fmt.Println("\nManual install instructions:")
			printManualInstall(certPath)
			return nil
		}

		fmt.Println("\nCA installed successfully!")
		fmt.Println("You can now use: saola proxy")
		return nil
	},
}

// installCA installs the CA cert into the OS trust store.
func installCA(certPath string) error {
	switch runtime.GOOS {
	case "darwin":
		// macOS: add to System Keychain.
		cmd := exec.Command("sudo", "security", "add-trusted-cert",
			"-d", "-r", "trustRoot",
			"-k", "/Library/Keychains/System.keychain",
			certPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "linux":
		// Linux: copy to ca-certificates and update.
		cmd := exec.Command("sudo", "cp", certPath, "/usr/local/share/ca-certificates/saola-proxy-ca.crt")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		cmd2 := exec.Command("sudo", "update-ca-certificates")
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		return cmd2.Run()

	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// printManualInstall prints manual trust instructions per OS.
func printManualInstall(certPath string) {
	switch runtime.GOOS {
	case "darwin":
		fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", certPath)
	case "linux":
		fmt.Printf("  sudo cp %s /usr/local/share/ca-certificates/saola-proxy-ca.crt\n", certPath)
		fmt.Println("  sudo update-ca-certificates")
	default:
		fmt.Printf("  Import %s into your system certificate store as a trusted root CA.\n", certPath)
	}
}

func init() {
	rootCmd.AddCommand(setupCACmd)
}
