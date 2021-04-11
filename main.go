package main

import (
	"bufio"
	"fmt"
	"github.com/cavaliercoder/grab"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/martinlindhe/inputbox"
	"github.com/sqweek/dialog"
	"ikamvope/pkg/icon"
	icon2 "ikamvope/pkg/icon_sync"
	"syscall"

	//"github.com/getlantern/systray/example/icon"
	"github.com/kennygrant/sanitize"
	"github.com/skratchdot/open-golang/open"
	"github.com/tebeka/selenium"
	"ikamvope/pkg/model"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	seleniumPath = `C:\SDKs\chromedriver_win32\chromedriver89.exe`
	port         = 9515
)

type FavoriteSite struct {
	id   string
	name string
}

var (
	wantsToSync      = false
	mSync            *systray.MenuItem
	studentNumber    string
	password         string
	baseDataPath     = "C:\\ikamva"
	favoriteSites    []FavoriteSite
	directories      []string
	downloadFileList []string
	ops              []selenium.ServiceOption
	webDriver        selenium.WebDriver
	err              error
	service          *selenium.Service
	cookieHttp       *http.Cookie
)

func HideConsole() {
	getConsoleWindow := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	if getConsoleWindow.Find() != nil {
		return
	}

	showWindow := syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	if showWindow.Find() != nil {
		return
	}

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		return
	}

	_, _, _ = showWindow.Call(hwnd, 0)
}

func onReady() {
	go func() {
		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("Ikamva Sync")
		systray.SetTooltip("Ikamva Sync - Idle")

		mLoggedIn := systray.AddMenuItem("Logged in as "+studentNumber, "")
		systray.AddSeparator()

		mOpenLocal := systray.AddMenuItem("Open Local", "Open folder containing synced items")
		mOpenIkamva := systray.AddMenuItem("Open Ikamva", "Open Ikamva in your web browser")
		mSync = systray.AddMenuItem("Sync", "Sync")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Exit", "")
		if wantsToSync == true {
			go sync()
			wantsToSync = false
		}

		for {
			select {
			case <-mOpenIkamva.ClickedCh:
				_ = open.Run("https://ikamva.uwc.ac.za/portal")
			case <-mOpenLocal.ClickedCh:
				_ = open.Run("C:\\ikamva")
			case <-mLoggedIn.ClickedCh:
				promptLoginDetails()
			case <-mSync.ClickedCh:
				go sync()
			case <-mQuit.ClickedCh:
				webDriver.Quit()
				systray.Quit()
				return
			}
		}
	}()
}

func sync() {
	err = beeep.Notify("Ikamva Sync", "Syncing", "assets/ik.png")
	if err != nil {
		panic(err)
	}
	systray.SetTemplateIcon(icon2.Data, icon2.Data)
	mSync.Disable()
	mSync.SetTitle("Syncing")
	systray.SetTooltip("Ikamva Sync - Syncing")
	syncIkamva()
	mSync.Enable()
	mSync.SetTitle("Sync")
	systray.SetTooltip("Ikamva Sync - Idle")
	systray.SetTemplateIcon(icon.Data, icon.Data)
	err = beeep.Notify("Ikamva Sync", "Sync complete", "assets/ik.png")
	if err != nil {
		panic(err)
	}
}
func syncIkamva() {
	service, err = selenium.NewChromeDriverService(seleniumPath, port, ops...)
	if err != nil {
		fmt.Printf("Error starting the ChromeDriver server: %v", err)
	}
	defer service.Stop()
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	//webDriver.ResizeWindow()
	webDriver, err = selenium.NewRemote(caps, "http://127.0.0.1:9515/wd/hub")
	if err != nil {
		panic(err)
	}
	defer webDriver.Quit()
	login()
	getFavoriteSites()
	for _, group := range favoriteSites {
		processGroup(group)
	}
	//getAnnouncements()

	webDriver.Quit()

	createDownloadDirectories()
	err = downloadBatch()
	if err != nil {
		panic(err)
	}

}

func promptLoginDetails() {
	u, ok := inputbox.InputBox("Ikamva Sync", "Enter Ikamva Username", studentNumber)
	if ok {
		if u == "" {
			os.Exit(0)
		}
		fmt.Println("username:", u)
		p, ok := inputbox.InputBox("Ikamva Sync", "Enter Ikamva Password", password)
		if ok {
			if p == "" {
				os.Exit(0)
			}
			fmt.Println("password:", p)
			err := os.MkdirAll(filepath.Join(baseDataPath), 0755)
			if err != nil {
				log.Fatal(err)
			}
			credentials := []string{u, p}
			file, err := os.OpenFile(filepath.Join(baseDataPath, ".credentials"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				log.Fatalf("failed creating file: %s", err)
			}

			datawriter := bufio.NewWriter(file)

			for _, data := range credentials {
				_, _ = datawriter.WriteString(data + "\n")
			}
			_ = datawriter.Flush()
			file.Close()
			readCredentials()
			ok := dialog.Message("%s", "Do you want to sync now?").Title("Ikamva Sync").YesNo()
			if ok {
				wantsToSync = true
				if mSync != nil {
					go sync()
				}
			}

		} else {
			promptLoginDetails()
		}
	} else {
		promptLoginDetails()
	}
}

func readCredentials() {
	fileIO, err := os.OpenFile(filepath.Join(baseDataPath, ".credentials"), os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	rawBytes, err := ioutil.ReadAll(fileIO)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(rawBytes), "\n")
	for i, line := range lines {
		if i == 0 {
			studentNumber = line
		}
		if i == 1 {
			password = line
		}

	}
	defer fileIO.Close()
}

func main() {

	HideConsole()
	readCredentials()
	if studentNumber == "" || password == "" {
		promptLoginDetails()

	}
	err := beeep.Notify("Ikamva Sync", "Running in the system tray", "assets/ik.png")
	if err != nil {
		panic(err)
	}

	onExit := func() {
		//now := time.Now()
		//ioutil.WriteFile(fmt.Sprintf(`on_exit_%d.txt`, now.UnixNano()), []byte(now.String()), 0644)
	}

	systray.Run(onReady, onExit)

}
func login() {
	if err := webDriver.Get("https://ikamva.uwc.ac.za/portal/site/test"); err != nil {
		panic(err)
	}
	we, err := webDriver.FindElement(selenium.ByID, `eid`)
	if err != nil {
		panic(err)
	}
	_ = we.SendKeys(studentNumber)
	we, err = webDriver.FindElement(selenium.ByID, `pw`)
	if err != nil {
		panic(err)
	}
	_ = we.SendKeys(password)
	we, err = webDriver.FindElement(selenium.ByID, `submit`)
	if err != nil {
		panic(err)
	}
	_ = we.Click()
	cookie, err := webDriver.GetCookie("JSESSIONID")
	if err != nil {
		panic(err)
	}
	cookieHttp = &http.Cookie{Name: cookie.Name, Value: cookie.Value}
}
func processGroup(site FavoriteSite) {
	groupBaseUrl := fmt.Sprintf("https://ikamva.uwc.ac.za/access/content/group/%s/", site.id)
	err = webDriver.Get(groupBaseUrl)
	if err != nil {
		panic(err)
	}
	getFiles()
	getFolders()
}

func getFiles() {
	files, err := webDriver.FindElements(selenium.ByCSSSelector, `body > div > ul > li.file > a`)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		fileHref, err := file.GetAttribute("href")
		if err != nil {
			panic(err)
		}
		addToDownloadList(fileHref)
	}
}

func getFavoriteSites() {
	req, err := http.NewRequest("GET", "https://ikamva.uwc.ac.za/portal/favorites/list", nil)
	if err != nil {
		return
	}
	req.AddCookie(cookieHttp)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatal("Status code != 200 for favorite site list")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	siteList, _ := model.UnmarshalSiteList(data)
	for _, id := range siteList.FavoriteSiteIDS {
		_ = webDriver.Get(fmt.Sprintf("https://ikamva.uwc.ac.za/portal/site/%s", id))
		titleAnchor, err := webDriver.FindElement(selenium.ByCSSSelector, "#topnav > li.Mrphs-sitesNav__menuitem.is-selected > a.link-container")
		if err != nil {
			panic(err)
		}
		name, err := titleAnchor.GetAttribute("title")
		if err != nil {
			panic(err)
		}
		favoriteSites = append(favoriteSites, FavoriteSite{
			id:   id,
			name: name,
		})
		_ = webDriver.Back()
	}
}

func getFolders() {
	folders, err := webDriver.FindElements(selenium.ByCSSSelector, `body > div > ul > li.folder > a`)
	if err != nil {
		panic(err)
	}
	var folderHrefs []string
	for _, folder := range folders {
		href, err := folder.GetAttribute("href")
		if err == nil {
			folderHrefs = append(folderHrefs, href)
		}
	}
	for _, href := range folderHrefs {
		fileUrl, _ := url.QueryUnescape(href)
		u, _ := url.ParseRequestURI(fileUrl)
		directory, _ := url.QueryUnescape(u.Path)
		directories = append(directories, directory)

		err = webDriver.Get(href)
		getFiles()
		getFolders()
		err = webDriver.Back()
		if err != nil {
			panic(err)
		}
	}
}

func addToDownloadList(href string) {
	fileUrl, _ := url.QueryUnescape(href)
	downloadFileList = append(downloadFileList, fileUrl)
}

func createDownloadDirectories() {
	for _, favoriteSite := range favoriteSites {
		err := os.MkdirAll(filepath.Join(baseDataPath, sanitizePath(favoriteSite.name)), 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, dir := range directories {
		err := os.MkdirAll(filepath.Join(baseDataPath, sanitizePath(dir)), 0755)
		if err != nil {
			log.Fatal(err)
		}

	}
}
func sanitizePath(path string) string {
	var result string
	for _, favoriteSite := range favoriteSites {
		if strings.Contains(path, favoriteSite.id) {
			result = strings.ReplaceAll(path, favoriteSite.id, favoriteSite.name)
			break
		}
	}
	result = strings.ReplaceAll(result, "https://ikamva.uwc.ac.za/access/content/group/", "")
	result = strings.ReplaceAll(result, "/access/content/group/", "")
	result = sanitize.Path(result)
	return result
}

func downloadBatch() error {
	reqs := make([]*grab.Request, 0)
	for _, urlPath := range downloadFileList {
		urlWebPath := filepath.Join(baseDataPath, sanitizePath(urlPath))
		req, err := grab.NewRequest(urlWebPath, urlPath)
		if err != nil {
			panic(err)
		}
		req.HTTPRequest.AddCookie(cookieHttp)
		reqs = append(reqs, req)
	}
	client := grab.NewClient()
	respch := client.DoBatch(4, reqs...)
	for resp := range respch {
		if err := resp.Err(); err != nil {
			panic(err)
		}

		fmt.Printf("Synced %s to %s\n", sanitizePath(resp.Request.URL().String()), resp.Filename)
	}
	return err
}
