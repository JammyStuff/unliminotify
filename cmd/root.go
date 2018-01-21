package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/jammystuff/gocineworld"
	"github.com/olekukonko/tablewriter"
	"github.com/sfreiberg/gotwilio"

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

	rootCmd.Flags().StringP("notifications-file", "f", "/var/db/unliminotify/notifications", "Path to the notifications file")
	viper.BindPFlag("notifications_file", rootCmd.Flags().Lookup("notifications-file"))

	rootCmd.Flags().IntP("cinema-id", "c", 1, "ID of the cinema to check")
	viper.BindPFlag("cinema_id", rootCmd.Flags().Lookup("cinema-id"))

	rootCmd.Flags().StringSliceP("sms-numbers", "s", []string{}, "Numbers to send SMS notifications to")
	viper.BindPFlag("sms_numbers", rootCmd.Flags().Lookup("sms-numbers"))

	rootCmd.Flags().Bool("disable-sms", false, "Disable SMS notification sending")
	rootCmd.Flags().MarkHidden("disable-sms")

	rootCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
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

func filterNewUnlimitedScreenings(films *[]gocineworld.Film) *[]gocineworld.Film {
	fmt.Print("Filtering out old Unlimited screenings... ")
	path := viper.GetString("notifications_file")
	rawFile, err := ioutil.ReadFile(path)
	if err != nil {
		rawFile = []byte{}
	}
	urls := strings.Split(string(rawFile), "\n")

	newScreenings := make([]gocineworld.Film, 0)
	for _, film := range *films {
		for _, show := range film.Shows {
			found := false
			for _, url := range urls {
				if show.URL == url {
					found = true
					break
				}
			}
			if !found {
				newScreenings = append(newScreenings, film)
				break
			}
		}
	}
	fmt.Println("OK")
	return &newScreenings
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

func getTwilioClient() *gotwilio.Twilio {
	sid := viper.GetString("twilio_sid")
	token := viper.GetString("twilio_token")
	return gotwilio.NewTwilioClient(sid, token)
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
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get verbose flag: %v", err)
	}

	disableSMS, err := cmd.Flags().GetBool("disable-sms")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get disable SMS flag: %v", err)
	}

	cinemaID := viper.GetInt("cinema_id")
	smsFrom := viper.GetString("twilio_from")
	smsNumbers := viper.GetStringSlice("sms_numbers")
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

	newScreenings := filterNewUnlimitedScreenings(unlimitedScrenings)
	if len(smsNumbers) != 0 {
		sendSMSNotifications(newScreenings, smsNumbers, smsFrom, disableSMS, verbose)
	}
	writeNotificationsFile(newScreenings)

	fmt.Print("\n")
	printUnlimitedScreenings(unlimitedScrenings)
}

func sendSMSNotifications(films *[]gocineworld.Film, numbers []string, smsFrom string, disableSMS, verbose bool) {
	fmt.Print("Sending SMS notifications... ")
	twilio := getTwilioClient()
	for _, film := range *films {
		title := film.Title
		for _, show := range film.Shows {
			datetime, err := show.Time()
			if err != nil {
				fmt.Println("ERROR")
				fmt.Fprintf(os.Stderr, "Error parsing show time: %v", err)
				os.Exit(1)
			}
			date := datetime.Format(dateFormat)
			time := datetime.Format(timeFormat)
			url := show.URL
			message := fmt.Sprintf("%s on %s @ %s: %s", smsFormatTitle(title), date, time, url)
			if verbose {
				fmt.Printf("\n%s\nSending SMS notifications... ", message)
			}
			if !disableSMS {
				for _, number := range numbers {
					resp, exception, err := twilio.SendSMS(smsFrom, number, message, "", "")
					if verbose {
						fmt.Printf("\nSMS response: %v\n", resp)
						fmt.Printf("\nSMS exception: %v\n", exception)
						fmt.Printf("\nSMS error: %v\n", err)
						fmt.Print("Sending SMS notifications...")
					}
					if exception != nil || err != nil {
						fmt.Println("ERROR")
						fmt.Fprintf(os.Stderr, "Error sending SMS notifications: %v %v", exception, err)
						os.Exit(1)
					}
				}
			}
		}
	}
	fmt.Println("OK")
}

func smsFormatTitle(title string) string {
	return strings.Replace(title, " : Unlimited Screening", "", 1)
}

func writeNotificationsFile(films *[]gocineworld.Film) {
	fmt.Print("Writing notifications file... ")
	path := viper.GetString("notifications_file")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Fprintf(os.Stderr, "Unable to open notifications file: %v", err)
		os.Exit(1)
	}
	defer file.Close()
	for _, film := range *films {
		for _, show := range film.Shows {
			_, err := fmt.Fprintln(file, show.URL)
			if err != nil {
				fmt.Println("ERROR")
				fmt.Fprintf(os.Stderr, "Unable to write to notifications file: %v", err)
				os.Exit(1)
			}
		}
	}
	fmt.Println("OK")
}
