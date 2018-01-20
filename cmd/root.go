package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/jammystuff/gocineworld"
	"github.com/olekukonko/tablewriter"

	"github.com/jammystuff/unliminotify/util"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const dateFormat = "Mon _2 Jan"
const timeFormat = "15:04"

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "unliminotify",
	Short: "Check for Cineworld Unlimited screenings",
	Long: `Checks which Cineworld Unlimited screenings appear in the current
listings and sends notifications for any that are new.`,
	Run: runRoot,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.unliminotify.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().IntP("cinema-id", "c", 1, "ID of the cinema to check")
	viper.BindPFlag("cinema_id", rootCmd.Flags().Lookup("cinema-id"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".unliminotify" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".unliminotify")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func findCinema(id int, listings *gocineworld.Listings) *gocineworld.Cinema {
	for _, cinema := range listings.Cinemas {
		if cinema.ID == id {
			return &cinema
		}
	}
	return nil
}

func findUnlimitedScreenings(films *[]gocineworld.Film) *[]gocineworld.Film {
	unlimitedScrenings := make([]gocineworld.Film, 0)
	for _, film := range *films {
		title := film.Title
		matched, err := regexp.Match("Unlimited Screening", []byte(title))
		if err != nil {
			fmt.Println("ERROR")
			fmt.Fprintf(os.Stderr, "Error checking if film is an Unlimited screening: %v", err)
			os.Exit(1)
		}
		if matched {
			unlimitedScrenings = append(unlimitedScrenings, film)
		}
	}
	return &unlimitedScrenings
}

func printUnlimitedScreenings(films *[]gocineworld.Film) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Title", "Date", "Time"})

	for _, film := range *films {
		title := film.Title
		for _, show := range film.Shows {
			datetime, err := show.Time()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing show time: %v", err)
				os.Exit(1)
			}
			date := datetime.Format(dateFormat)
			time := datetime.Format(timeFormat)
			table.Append([]string{title, date, time})
		}
	}

	table.Render()
}

func runRoot(cmd *cobra.Command, args []string) {
	cinemaID := viper.GetInt("cinema_id")
	listingsXML := util.FetchListingsXML()
	listings := util.ParseListingsXML(listingsXML)

	fmt.Print("Finding cinema... ")
	cinema := findCinema(cinemaID, &listings)
	if cinema == nil {
		fmt.Println("ERROR")
		fmt.Fprintf(os.Stderr, "Unable to find cinema %d", cinemaID)
		os.Exit(1)
	}
	fmt.Println("OK")

	fmt.Printf("Checking for Unlimited screenings at %s... ", cinema.Name())
	unlimitedScrenings := findUnlimitedScreenings(&cinema.Films)
	fmt.Println("OK")

	fmt.Print("\n")
	printUnlimitedScreenings(unlimitedScrenings)
}
