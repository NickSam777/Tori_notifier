package main

import (
	"bufio"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/Syfaro/telegram-bot-api"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mutex sync.Mutex

var data_folder = os.Args[1] + "/"
var token = os.Args[2]
var checking_interval, _ = strconv.Atoi(os.Args[3])

func _check(err error) {
	if err != nil {
		panic(err)
	}
}

func createFile(path string) {
	// check if file exists

	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		_check(err)

		defer file.Close()
	}

	fmt.Println("File Created Successfully", path)
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func Contains(a []string, x string) bool {
	for _, n := range a {
				n = strings.TrimSpace(n)
		if x == n {
			return true
		}
	}
	return false
}

func telegramBot() {

	bot, err := tgbotapi.NewBotAPI(token)

	_check(err)

	u := tgbotapi.NewUpdate(0)

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Make sure that message in text
		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {

			chat_folder := data_folder + strconv.FormatInt(update.Message.Chat.ID, 10)

			switch update.Message.Text {
			case "/start":

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, i'm a Tori notification bot!")
				bot.Send(msg)
				msg1 := tgbotapi.NewMessage(update.Message.Chat.ID, "Send me Tori advertisements list URL sorted by newest to start receiving notifications.")
				bot.Send(msg1)
				msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "To stop receiving notifications send me /stop")
				bot.Send(msg2)

				if os.Getenv("NOTIFY_TO_CHAT") != "" {
					chat_id_int, err := strconv.ParseInt(os.Getenv("NOTIFY_TO_CHAT"), 10, 64)
					_check(err)

					msg := tgbotapi.NewMessage(chat_id_int, "New user: @"+update.Message.From.UserName)
					bot.Send(msg)
				}

				fmt.Println("Start chat with id:" + strconv.FormatInt(update.Message.Chat.ID, 10) + ". User: @" + update.Message.From.UserName)

			case "/stop":
				err := os.RemoveAll(chat_folder)
				_check(err)

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You successfully stop following all advertisements")
				bot.Send(msg)

			default:

				url := update.Message.Text

				fmt.Println("request: " + url)

				// Request the HTML page.
				res, err := http.Get(url)

				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "URL is wrong"))
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Correct URL sample: https://www.tori.fi/koko_suomi/tietokoneet_ja_lisalaitteet/verkkotuotteet?ca=18&st=s&st=k&st=u&st=h&st=g&st=b&w=3&cg=5030&c=5039"))
					continue
				}

				defer res.Body.Close()
				if res.StatusCode != 200 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "URL is wrong"))
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Correct URL sample: https://www.tori.fi/koko_suomi/tietokoneet_ja_lisalaitteet/verkkotuotteet?ca=18&st=s&st=k&st=u&st=h&st=g&st=b&w=3&cg=5030&c=5039"))
					continue
				}

				// Load the HTML document
				doc, err := goquery.NewDocumentFromReader(res.Body)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "URL is wrong"))
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Correct URL sample: https://www.tori.fi/koko_suomi/tietokoneet_ja_lisalaitteet/verkkotuotteet?ca=18&st=s&st=k&st=u&st=h&st=g&st=b&w=3&cg=5030&c=5039"))
					continue
				}

				if len(doc.Find("a[id*=item_]").Nodes) == 0 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "URL is wrong"))
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Correct URL sample: https://www.tori.fi/koko_suomi/tietokoneet_ja_lisalaitteet/verkkotuotteet?ca=18&st=s&st=k&st=u&st=h&st=g&st=b&w=3&cg=5030&c=5039"))
					continue
				}

				// Check if URL allready exists
				advertisements_path := chat_folder + "/advertisements"
				if _, err := os.Stat(advertisements_path); err == nil {

					lines, err := readLines(advertisements_path)
					_check(err)
					if Contains(lines, url) {

						bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "This URL allready exists"))
						continue
					}
				}


				// Write url to advertisements list
				os.MkdirAll(chat_folder, os.ModePerm)
				file_adv, err := os.OpenFile(advertisements_path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
				_check(err)
				file_adv.WriteString(url + " \n")
				defer file_adv.Close()

				// Create "sended_links" if not exists
				file_lnk, err := os.OpenFile(chat_folder+"/sended_links", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
				_check(err)
				defer file_lnk.Close()

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Now you start following this url: "+url)
				bot.Send(msg)
				check_updates(false)
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send URL for subscribe")
			bot.Send(msg)

		}
	}
}

func check_updates(notify bool) {
	mutex.Lock()
	defer mutex.Unlock()

	bot, err := tgbotapi.NewBotAPI(token)

	_check(err)

	folders, err := ioutil.ReadDir(data_folder)
	_check(err)

	for _, folder := range folders {
		if folder.IsDir() {
			chat_id := folder.Name()

			advertisements, err := readLines(data_folder + chat_id + "/advertisements")
			_check(err)

			for _, url := range advertisements {

				url = strings.TrimSpace(url)
				sended_links_path := data_folder + chat_id + "/sended_links"

				lines, err := readLines(sended_links_path)
				_check(err)

				doc, err := goquery.NewDocument(url)

				// Links
				var aLink = make([]string, 0)
				// First element
				doc.Find("a[tabindex]").Each(func(i int, s *goquery.Selection) {

					link, _ := s.Attr("href")
					link = strings.TrimSpace(link)
					aLink = append(aLink, link)
				})

				doc.Find("a[id*=item_]").Each(func(i int, s *goquery.Selection) {

					link, _ := s.Attr("href")

					aLink = append(aLink, link)
				})

				// Prices
				var aPrice = make([]string, 0)
				doc.Find("p[class*=list_price]").Each(func(i int, s *goquery.Selection) {

					price := s.Text()
					aPrice = append(aPrice, price)
				})

				// Images
				var aImage = make([]string, 0)
				doc.Find("img[class=item_image]").Each(func(i int, s *goquery.Selection) {

					image, _ := s.Attr("src")
					aImage = append(aImage, image)

					//fmt.Printf(image)
					//fmt.Scanln()

				})

				//fmt.Println(len(aLink), len(aPrice), len(aImage))
				//fmt.Scanln()

				for i, link := range aLink {

					//fmt.Println(i, link, aPrice[i])
					//fmt.Scanln()

					if !Contains(lines, link) {

						lines = append(lines, link)

						if notify {

							chat_id_int, err := strconv.ParseInt(chat_id, 10, 64)
							_check(err)

							// Send Link
							msg := tgbotapi.NewMessage(chat_id_int, link)
							bot.Send(msg)

							// Send Image
							if i < len(aImage) {
								http.Get("https://api.telegram.org/bot" + token + "/sendPhoto?chat_id=" + chat_id + "&photo=" + aImage[i])
							}

							// Send Price
							if i < len(aImage) {
								msg = tgbotapi.NewMessage(chat_id_int, "<b>Price </b><b>"+aPrice[i]+"</b>")
								msg.ParseMode = "HTML"
								bot.Send(msg)
								fmt.Println(i, link, aPrice[i])
							}

						}
					}

				}

				err = writeLines(lines, sended_links_path)
				_check(err)

			}
		}
	}
}

func main() {

	go telegramBot()

	for {
		check_updates(true)
		time.Sleep(time.Second * time.Duration(checking_interval))
	}
}

// https://pkg.go.dev/gopkg.in/telegram-bot-api.v4

//https://api.telegram.org/bot6401632822:AAEsk9i4wz6_XmLhIauxVWopcwVOnaEkme4/sendPhoto?chat_id=1922860830&photo=https://images.tori.fi/api/v1/imagestori/images/100233627540.jpg?rule=thumb_280x210

