package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"bytes"
)

var (
	addr          = flag.String("addr", ":8080", "http service address")
	assets        = flag.String("assets", defaultAssetPath(), "path to assets")
	homeTempl     *template.Template
	editPlayerTempl     *template.Template
	playerIdTempl     *template.Template
	places        []Place
	chars         []Char
	npcs          []Char
	place         = "void"
	NoText        bool
	ShowParty     bool
	ShowNpcs      bool
	ShowMugs      bool
	initiativetxt string
	curhps        map[int]int
	lastoutput    string
	lastbattlemsg string
	battlelog     string
	outputar      []string
	currentturn   int
	objects       []Object
	scenes        []Scene
	scene         string
	players	      []string
	loggedin      []string
	charlist	string
)

type MainData struct {
	Host    string
	Content string
}

type Command struct {
	Name    string
	Args    []string
	RawArgs string
}

type Place struct {
	Name     string
	Image    string
	Desc     string
	Key      string
	Autodrop []string
}

type Object struct {
	Key  string
	Name string
	Image string
	Desc string
	Contains []string
	Weight int
}

type Scene struct {
	Key     string
	Desc    string
	Chars   []string
	Objects []Placement
}

type Placement struct {
	ObjKey  string
	Context string
}

type Char struct {
	// Required fields for char or npcs
	Name       string
	Class      string
	Race       string
	Abilities  Abilities
	Level      int
	InParty    bool
	Image      string
	Initiative int
	AC         int
	Alignment  string
	HP         int
	Desc       string
	Key        string
	Attacks    []Attack
	Inventory  []string


	// extra fields for player character
	Inspiration int
	Speed int
	Skills string
	ProfBonus int
	PassPerception int
	MiscProfLanguages string
	HitDice string
	PersonalityTraits string
	Background string
	Ideals string
	Bonds string
	Flaws string
	FeaturesTraits string
	Treasure string
	SpellL0 string
	SpellL1 string
	SpellL2 string
	SpellL3 string
	SpellL4 string
	SpellL5 string
	SpellL6 string
	SpellL7 string
	SpellL8 string
	SpellL9 string
	AttacksJson string
	AbilitiesJson string
	InventoryJson string
	Playername string
}

type Attack struct {
	Name       string
	Range      string
	Dtype      string
	Verb       string
	Hitbonus   int
	Damageroll string
}

type Abilities struct {
	Str int
	Dex int
	Con int
	Int int
	Wis int
	Cha int
}

func (char *Char) JsonIfy() {
	jsonatt,_ := json.Marshal(char.Attacks)
	jsoninv,_ := json.Marshal(char.Inventory)
	jsonabi,_ := json.Marshal(char.Abilities)
	char.AttacksJson = template.HTMLEscapeString(string(jsonatt))
	char.InventoryJson = template.HTMLEscapeString(string(jsoninv))
	char.AbilitiesJson = template.HTMLEscapeString(string(jsonabi))

	//fmt.Println(char.AttacksJson)
}

func (att *Attack) Display() string {
	output := ""

	output = fmt.Sprintf("<td>%s</td> <td>%d</td> <td>%s</td> <td>%s</td><td>%s</td>", att.Name, att.Hitbonus, att.Damageroll, att.Range, att.Dtype)

	return output
}

func getPlace(name string) Place {
	for i := range places {
		if places[i].Name == name || places[i].Key == name {
			return places[i]
		}
	}

	return Place{}
}

func getScene(key string) (Scene, int) {
	for i := range scenes {
		if scenes[i].Key == key {
			return scenes[i], i
		}
	}

	return Scene{}, 0
}

func getObject(key string) Object {
	for i := range objects {
		if objects[i].Key == key {
			return objects[i]
		}
	}

	return Object{}
}

func getChar(name string) Char {
	for i := range chars {
		//prefix := strings.ToLower(chars[i].Name[0:3])
		if chars[i].Name == name {
			return chars[i]
		}

		if chars[i].Key == name {
			return chars[i]
		}

	}

	return Char{}
}

func prefixExists(prefix string) bool {
	for i := range chars {
		if chars[i].Key == prefix {
			return true
		}
	}
	return false
}

func makeCharKey(name string) string {

	prefix := strings.ToLower(name[0:3])
	prefixOrig := strings.ToLower(name[0:3])
	prefixInc := 1
	for {
		if prefixExists(prefix) {
			prefix = fmt.Sprintf("%s%d", prefixOrig, prefixInc)
			prefixInc = prefixInc + 1
			continue
		}
		return prefix
	}

}

func dropItems(char Char) {
	//fmt.Println(char.Name, "...", char.Inventory)
	_, indx := getScene(scene)
	//fmt.Println(scenes[indx].Objects)
	for i := range char.Inventory {
		obj := getObject(char.Inventory[i])
		fmt.Println("dropped ", obj.Key)
		scenes[indx].Objects = append(scenes[indx].Objects, Placement{ObjKey: obj.Key, Context: fmt.Sprintf(" on the corpse of %s", char.Name)})
	}
	//fmt.Println(scenes[indx].Objects)
}

func applyDamage(char Char, damage int) {
	if charIsNpc(char.Name) {
		for i := range npcs {
			if char.Key == npcs[i].Key {
				npcs[i].HP = npcs[i].HP - damage
				//fmt.Println("Calling dropItems...")
				if npcs[i].HP <= 0 {
					dropItems(npcs[i])
				}
			}
		}
	} else {
		for i := range chars {
			if chars[i].Name == char.Name {
				curhps[i] = curhps[i] - damage
			}
		}
	}
}

func ReadFileContents(file *os.File) []byte {
	reader := bufio.NewReader(file)
	rawBytes, _ := ioutil.ReadAll(reader)
	return rawBytes
}

func initPlaces() {

	file, err := os.Open("assets/places.json")

	if err != nil {
		panic("Could not open assets/places.json")
	}

	filebytes := ReadFileContents(file)

	places = make([]Place, 30)
	err = json.Unmarshal(filebytes, &places)
	if err != nil {
		fmt.Println("Failed to read assets/places.json: ", err)
		panic(err)
	}

	//return *places
}

func initScenes() {
	file, err := os.Open("assets/scenes.json")

	if err != nil {
		panic("Could not open assets/scenes.json")
	}

	filebytes := ReadFileContents(file)

	//if scenes == nil {
		scenes = make([]Scene, 20)
	//}

	err = json.Unmarshal(filebytes, &scenes)
	if err != nil {
		fmt.Println("Failed to read assets/scenes.json: ", err)
		panic(err)
	}

}

func initObjects() {
	file, err := os.Open("assets/objects.json")

	if err != nil {
		panic("Could not open assets/objects.json")
	}

	filebytes := ReadFileContents(file)

	//if objects == nil {
	objects = make([]Object, 30)
	//}

	err = json.Unmarshal(filebytes, &objects)
	if err != nil {
		fmt.Println("Failed to read assets/objects.json: ", err)
		panic(err)
	}

}

func listPlaces() {

	for i := range places {
		fmt.Println(places[i].Key, ": ", places[i].Name, " - ", places[i].Desc)
	}

}

func listScenes() {

	for i := range scenes {
		fmt.Println(scenes[i].Key, ": ", scenes[i].Desc, scenes[i].Chars, scenes[i].Objects)
	}

}

func listObjs() {

	for i := range objects {
		fmt.Println(objects[i].Key, ": ", objects[i].Name, " - ", objects[i].Desc)
	}

}





func listChars() {

	for i := range chars {
		fmt.Println(chars[i].Key, chars[i].Name, ": ", chars[i].Desc)
	}
}

func listNpcs() {

	for i := range npcs {
		fmt.Printf("(%s) %s: %d\n", npcs[i].Key, npcs[i].Name, npcs[i].HP)
	}
}

func cloneChar(char Char) Char {
	nchar := Char{}

	nchar.Name = char.Name
	nchar.Class = char.Class
	nchar.Race = char.Race
	nchar.Abilities = char.Abilities
	nchar.Level = char.Level
	nchar.InParty = char.InParty
	nchar.Image = char.Image
	nchar.Initiative = char.Initiative
	nchar.AC = char.AC
	nchar.Alignment = char.Alignment
	nchar.HP = char.HP
	nchar.Desc = char.Desc
	nchar.Key = makeCharKey(char.Name)
	nchar.Attacks = char.Attacks
	nchar.Inventory = char.Inventory

	return nchar
}

func initChars(wipehps bool) {

	file, err := os.Open("assets/chars.json")

	if err != nil {
		panic("Could not open assets/chars.json")
	}

	filebytes := ReadFileContents(file)
	//if chars == nil {
	chars = make([]Char, 30)
	//nchars = make([]Char,30)
	//}
	err = json.Unmarshal(filebytes, &chars)
	if err != nil {
		fmt.Println("Failed to read assets/chars.json: ", err)
		panic(err)
	}

	for i := range players {
		char := loadPlayerChar(players[i])
		char.Playername = players[i]
		fmt.Println("Loaded ", char.Name)
		chars = append(chars,char)
	}

	//if curhps != nil {
	//	curhps = make(map[int]int)
	//}

	for i := range chars {
		if wipehps {
			curhps[i] = chars[i].HP
		}
		chars[i].Key = makeCharKey(chars[i].Name)

		/*if chars[i].NpcInstances > 0 {
			for k := 0; k <= chars[i].NpcInstances; k++ {
				nchars[k
			}
		}*/
	}

}

func initNpcs() {
	npcs = nil
	//npcs = new([]Char)
}

func defaultAssetPath() string {
	p, err := build.Default.Import(".", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

func alreadyLoggedIn(playername string) bool {
	for i := range loggedin {
		if loggedin[i] == playername {
			return true
		}
	}

	return false
}

func validatePlayer(player string) bool {
	if player == "ohgodmedusa" {
		return true
	}

	for i := range players {
		if strings.ToLower(player) == players[i] {
			//if alreadyLoggedIn(players[i]) {
			//	fmt.Println("Already logged in! ", players[i])
			//	return false
			//}
			fmt.Println("Fantastic, you are", player)
			loggedin = append(loggedin,players[i])
			return true
		}
	}

	return false
}

func homeHandler(c http.ResponseWriter, req *http.Request) {

	req.ParseForm()
	cv,_ := req.Cookie("playername")
	//if cv != nil && cv.Value != "" && !alreadyLoggedIn(cv.Value) {
	//		http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
	//}

	if (cv == nil || cv.Value == "") && len(req.Form["playername"]) == 0 {
		playerIdTempl.Execute(c,nil)
		return
	}


	if strings.Contains(req.URL.Path, "assets") {
		//fmt.Println("Serving assets...")
		chttp.ServeHTTP(c, req)

	} else if strings.Contains(req.URL.Path, "logout") {
		http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
		http.Redirect(c,req,"/",302)
	} else if strings.Contains(req.URL.Path, "char") {
		webViewChar(c, req)
	} else if strings.Contains(req.URL.Path, "playeredit") {
		fmt.Println("Player edit...")
		editPlayerChar(c, req)
	} else if strings.Contains(req.URL.Path, "playerid") {
		if len(req.Form["playername"]) == 1 && validatePlayer(req.Form["playername"][0]) {
			http.SetCookie(c,&http.Cookie{Name: "playername", Value: req.Form["playername"][0]})
			http.Redirect(c,req,"/",302)
			fmt.Println("Got cookie, redirecting.")
			//mainData := MainData{Host: req.Host, Content: lastoutput}
			//homeTempl.Execute(c, mainData)
		} else if cv != nil && cv.Value != "" {
			http.Redirect(c,req,"/",302)
			//mainData := MainData{Host: req.Host, Content: fmt.Sprintf("You are already logged in as %s", cv.Value)}
			//homeTempl.Execute(c, mainData)
		} else {
			//http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
			//http.Redirect(c,req,"/",302)
			playerIdTempl.Execute(c,nil)
		}
	} else {
		mainData := MainData{Host: req.Host, Content: lastoutput}
		homeTempl.Execute(c, mainData)
	}

}

func parseInput(input string) *Command {
	input = strings.TrimRight(input, "\n")
	parts := strings.Split(input, " ")
	cmd := &Command{}
	if len(parts) < 1 {
		fmt.Println("Invalid command.")
		return &Command{Name: "Invalid command"}
	} else if len(parts) >= 1 {
		cmd = &Command{Name: parts[0], Args: parts[1:]}
		cmd.RawArgs = strings.Join(parts[1:], " ")
	}

	//fmt.Println(cmd)
	//fmt.Println(cmd.Name)
	return cmd
}

func getNpcTxt() string {
	output := ""

	if !ShowNpcs {
		return output
	}

	for i := range npcs {
		if npcs[i].HP <= 0 {
			npcs[i].Image = "/assets/skull.jpg"
		}
		if ShowMugs {
			output = output + fmt.Sprintf("<div class=\"npc\"><a href=\"/char?name=%s\"><img src=\"%s\" width=120/></a><br><b>%s (%s)</b><br>%s</div>  ", npcs[i].Name, npcs[i].Image, npcs[i].Name, npcs[i].Key, npcs[i].Race)
		} else {
			output = output + fmt.Sprintf("<div class=\"npcnoimg\"><b>%s</b><br>%s   </div>", npcs[i].Name, npcs[i].Race)
		}
	}

	//fmt.Println(output)
	return output

}

func renderParty() string {

	output := ""

	if !ShowParty {
		return output
	}

	for i := range chars {
		curhp := curhps[i]
		if chars[i].InParty {

			if curhp < 0 {
				chars[i].Image = "/assets/skull.jpg"
			}

			if ShowMugs {
				output = output + fmt.Sprintf("<div id=\"%s\" class=\"partymember\"><div><a href=\"/char?name=%s\"><img src=\"%s\" width=120/></a></div><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", chars[i].Name, chars[i].Name, chars[i].Image, chars[i].Name, chars[i].Race, chars[i].Class, chars[i].Level, curhp, chars[i].HP)
			} else {
				output = output + fmt.Sprintf("<div id=\"%s\" class=\"partymembernoimg\"><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", chars[i].Name, chars[i].Name, chars[i].Race, chars[i].Class, chars[i].Level, curhp, chars[i].HP)
			}
		}
	}

	return output
}

func renderContent(msg string, cmd *Command) string {
	cplace := getPlace(place)
	imagetxt := "<script type=\"text/javascript\">$(\"#picture\").text(\"\");"
	if cplace.Image != "" {
		imagetxt = fmt.Sprintf("<script type=\"text/javascript\">$(\"#picture\").text(\"\");$(\"#picture\").append(\"<img src='%s'/>\");", cplace.Image)
	}

	if initiativetxt != "" && cmd.Name != "v" && cmd.Name != "vo" && cmd.Name != "blog" {
		initiativetxt = renderInitiativeTxt(outputar)
		msg = fmt.Sprintf("<div id=\"initiative\"><span id=\"initiativetxt\">%s</span></div><div id=\"msgtxt\">%s</div>", initiativetxt, msg)
	}

	placedesc := cplace.Desc
	if scene != "" {
		cscene, _ := getScene(scene)
		placedesc = cplace.Desc + "<br>" + cscene.Desc
		if len(cscene.Objects) != 0 {
			placedesc = placedesc + "<br>Visible objects: "
			for i := range cscene.Objects {
				placedesc = placedesc + "<br>" + getObject(cscene.Objects[i].ObjKey).Name + cscene.Objects[i].Context + "(" + getObject(cscene.Objects[i].ObjKey).Key + ")"
			}
		}
	}
	npctxt := getNpcTxt()
	content := fmt.Sprintf("<div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"desc\">%s</div><div id=\"msg\">%s</div><div id=\"npcs\">%s</div>  </div>    <div id=\"party\"><div id=\"partyinner\">%s</div></div>  %s$(\"#picture\").css(\"opacity\", \".37\");</script>", cplace.Name, placedesc, msg, npctxt, renderParty(), imagetxt)
	if NoText {
		content = fmt.Sprintf("%s$(\"#picture\").css(\"opacity\", \"1\");</script>", imagetxt)
	}

	if cmd.Name == "v" || cmd.Name == "vo" || cmd.Name == "blog" {
		content = fmt.Sprintf("<div id=\"msg\">%s</msg>", msg)
	}

	// DEBUG
	//fmt.Println(content)
	return content
}

func getDiceResults(dicestring string) string {
	cmd := exec.Command("rolldice", dicestring)
	results, err := cmd.Output()
	if err != nil {
		fmt.Println(fmt.Sprintf("rolldice %s", dicestring), results, " with err ", err)
	}
	//fmt.Println(dicestring,"...",string(results))
	return string(results)
}

func charIsNpc(name string) bool {
	for i := range npcs {
		if npcs[i].Name == name || npcs[i].Key == name {
			return true
		}
	}

	return false
}

func renderInitiativeTxt(ar []string) string {
	output := ""
	for i := range outputar {
		if ar[i] == "" {
			continue;
		}
		if i == currentturn {
			output = fmt.Sprintf("%s<span id=\"currentturn\">%s</span><br>", output, ar[i])
		} else {
			output = fmt.Sprintf("%s%s<br>", output, ar[i])
		}
	}


	return output
}

func rollInitiatives(advantages string) string {
	//output := "<b>Initiative</b>"
	output := ""

	advs := strings.Split(advantages, " ")

	rolls := make(map[int]int)
	var values []int
	//var outputar []string
	csize := 0
	for i := range chars {
		//fmt.Println(i, " for ", chars[i].Name)
		if chars[i].InParty || charIsNpc(chars[i].Name) {
			//fmt.Println("Rolling init for...", chars[i].Name)
			csize++
			//dres := getDiceResults(fmt.Sprintf("1d20+%d",chars[i].Initiative))
			dres := getDiceResults("1d20")
			dres = strings.TrimRight(dres, " \n")
			init, err := strconv.Atoi(dres)
			init = init + chars[i].Initiative

			for k := range advs {
				ap := strings.Split(advs[k], "=")
				if len(ap) < 2 {
					break
				}

				if ap[0] == chars[i].Key {
					dres2 := getDiceResults("1d20")
					dres2 = strings.TrimRight(dres2, " \n")
					init2, _ := strconv.Atoi(dres2)
					init2 = init2 + chars[i].Initiative
					if ap[1] == "adv" {
						fmt.Printf("%s advantage %d %d\n", chars[i].Name, init, init2)
						if init2 >= init {
							init = init2
						}
					} else if ap[1] == "dis" {
						fmt.Printf("%s disadvantage %d %d\n", chars[i].Name, init, init2)
						if init2 <= init {
							init = init2
						}
					}
					break
				}
			}

			if err != nil {
				fmt.Println("Could not convert ", dres, " to int: ", err)
			}
			rolls[i] = init
			values = append(values, init)
		}
	}

	outputar = make([]string, 1)
	sort.Sort(sort.Reverse(sort.IntSlice(values)))
	for i := range values {
	CHAR:
		for k := range chars {
			if (chars[k].InParty || charIsNpc(chars[k].Name)) && rolls[k] == values[i] {
				for j := range outputar {
					if strings.Contains(outputar[j], chars[k].Name) {
						continue CHAR
					}
				}
				outputar = append(outputar, fmt.Sprintf("%s (%d)", chars[k].Name, values[i]))
				fmt.Printf("%s - %s (%d)\n", chars[k].Key, chars[k].Name, values[i])
				//output = fmt.Sprintf("%s<br>%s (%d)",output,chars[k].Name, values[i])
			}
		}
	}

	currentturn = 0
	nextTurn()
	output = renderInitiativeTxt(outputar)

	return output
}

func attack(char1 Char, atti int, char2 Char, adv string) string {
	if atti >= len(char1.Attacks) {
		fmt.Println("Invalid attack index ", atti)
		return "";
	}
	genericMsg := fmt.Sprintf("%s attacks %s with %s<br>\n", char1.Name, char2.Name, char1.Attacks[atti].Name)
	//hitroll := getDiceResults(fmt.Sprintf("1d20+%d",char1.Attacks[atti].Hitbonus))
	attackstring := ""

	hitroll := getDiceResults("1d20")
	hitroll = strings.TrimRight(hitroll, " \n")
	hr, _ := strconv.Atoi(hitroll)
	hrb := hr + char1.Attacks[atti].Hitbonus
	hitroll2 := getDiceResults("1d20")
	hitroll2 = strings.TrimRight(hitroll2, " \n")
	hr2, _ := strconv.Atoi(hitroll2)
	hrb2 := hr2 + char1.Attacks[atti].Hitbonus
	if adv == "adv" {
		attackstring = fmt.Sprintf("Advantage attack roll: 1d20+%d (%d %d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, hrb2, char2.AC)
		if hr2 >= hr {
			hr = hr2
			hrb = hrb2
		}
	} else if adv == "dis" {
		attackstring = fmt.Sprintf("Disadvantage attack roll: 1d20+%d (%d %d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, hrb2, char2.AC)
		if hr2 <= hr {
			hr = hr2
			hrb = hrb2
		}
	} else {
		attackstring = fmt.Sprintf("Attack roll: 1d20+%d (%d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, char2.AC)
	}

	damageroll := getDiceResults(char1.Attacks[atti].Damageroll)
	damageroll = strings.TrimRight(damageroll, " \n")
	dr, _ := strconv.Atoi(damageroll)
	battlemsg := ""
	if hr == 20 {
		applyDamage(char2, dr)
		attackstring = attackstring + " <b>CRITICAL HIT!</b><br>\n"
		damagestring := fmt.Sprintf("%s savagely %s %s for %d damage.<br>\nDamage Roll: %s (%d)", char1.Name, char1.Attacks[atti].Verb, char2.Name, dr, char1.Attacks[atti].Damageroll, dr)
		battlemsg = genericMsg + attackstring + damagestring
	} else if hr == 1 {
		battlemsg = genericMsg + attackstring + " <b> CRITICAL MISS!</b>\n"
	} else if hrb >= char2.AC {
		applyDamage(char2, dr)
		attackstring = attackstring + " <b>HIT!</b><br>\n"
		damagestring := fmt.Sprintf("%s %s %s for %d damage.<br>\nDamage Roll: %s (%d)", char1.Name, char1.Attacks[atti].Verb, char2.Name, dr, char1.Attacks[atti].Damageroll, dr)
		battlemsg = genericMsg + attackstring + damagestring
	} else {
		battlemsg = genericMsg + attackstring + " <b>MISS!</b>"
	}

	fmt.Println(battlemsg)
	lastbattlemsg = ""
	fbattlemsg := fmt.Sprintf("<span class=\"blog\">%s</span><br>", lastbattlemsg) + battlemsg
	lastbattlemsg = battlemsg
	battlelog = battlelog + "<br>" + battlemsg
	return fbattlemsg
}

func viewObj(key string) string {
	output := ""
	obj := getObject(key)
	output = fmt.Sprintf("<div id=\"viewobj\"><img id=\"objimg\" src=\"%s\"/></div><div id=\"objinfo\"><p>%s</p><p>%s</p>",  obj.Image, obj.Name, obj.Desc)
	if len(obj.Contains) != 0 {
		output = output + "<br>Contains:<br>"
		for i := range obj.Contains {
			obj2 := getObject(obj.Contains[i])
			output = output + obj2.Name + " (" + obj2.Key + ")<br>"
		}

	}
	return output
}

func viewChar(name string, showstats bool) string {
	output := ""
	char := getChar(name)

	attacks := "<table><tr><td>name</td>  <td>to hit</td>  <td>dmg</td>  <td>range</td><td>type</td></tr>"
	for i := range char.Attacks {
		attacks = attacks + "<tr>\n" + char.Attacks[i].Display() + "</tr>"
	}
	attacks = attacks + "</table>"

	inventory := "<br>Inventory<br>"
	for i := range char.Inventory {
		obj := getObject(char.Inventory[i])
		inventory = inventory + obj.Name + " (" + obj.Key + ")<br>"
	}

	if showstats {

		output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s, Level %d %s</p><p>%s</p>Str: %d Dex: %d Con: %d Int: %d Wis: %d Cha: %d<br>Initiative: %d<br>AC: %d<br>HP: %d<br>Alignment: %s<br>Attacks:<br> %s%s", char.Image, char.Name, char.Level, char.Class, char.Desc, char.Abilities.Str, char.Abilities.Dex, char.Abilities.Con, char.Abilities.Int, char.Abilities.Wis, char.Abilities.Cha, char.Initiative, char.AC, char.HP, char.Alignment, attacks, inventory)
		output = output + fmt.Sprintf("<hr>Inspiration: %d<br> Proficiency Bonus: %d<br> Passive Perception: %d<br> Hit Dice: %s<br> Speed: %d<br> Skills: %s<br><hr>Misc Proficiencies and Languages: %s<br> Personality Traits: %s<br> Ideals: %s<br> Bonds: %s<br> Flaws: %s<br> Features and Traits: %s<br> Treasure: %s<br> Spells<br> Level 0: %s<br> Level 1: %s<br> Level 2: %s<br> Level 3: %s<br> Level 4: %s<br> Level 5: %s<br> Level 6: %s<br> Level 7: %s<br> Level 8: %s<br> Level 9: %s<br> </div>", char.Inspiration, char.ProfBonus, char.PassPerception, char.HitDice, char.Speed, char.Skills, char.MiscProfLanguages, char.PersonalityTraits, char.Ideals, char.Bonds, char.Flaws, char.FeaturesTraits, char.Treasure, char.SpellL0, char.SpellL1, char.SpellL2, char.SpellL3, char.SpellL4, char.SpellL5, char.SpellL6, char.SpellL7, char.SpellL8, char.SpellL9)
	} else {
		output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s</p><p>%s</p>", char.Image, char.Name, char.Desc)
	}

	//fmt.Println(char.Name)
	//fmt.Println(output)
	return output
}


/*
func viewNpcChar(name string) string {
	output := ""
	char := getChar(name)

/*	attacks := "<table><tr><td>name</td>  <td>to hit</td>  <td>dmg</td>  <td>type</td></tr>"
	for i := range char.Attacks {
		attacks = attacks + "<tr>\n" + char.Attacks[i].Display() + "</tr>"
	}
	attacks = attacks + "</table>"
	*/

//	output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s</p><p>%s</div>", char.Image, char.Name, char.Desc)

	//fmt.Println(char.Name)
	//fmt.Println(output)
//	return output
//}

func dropNpc(name string) {
	nchar := cloneChar(getChar(name))
	chars = append(chars, nchar)
	npcs = append(npcs, nchar)
	fmt.Println("Dropped ", nchar.Key)
}

func syncNpcs() {
	for i := range npcs {
		chars = append(chars,npcs[i])
	}
}

func setHP(name string, hp int) {
	if charIsNpc(name) {
		for i := range npcs {
			if name == npcs[i].Key {
				npcs[i].HP = hp
				fmt.Println(hp)
				if hp <= 0 {
					dropItems(npcs[i])
				}
			}
		}
	} else {
		for i := range chars {
			if chars[i].Key == name {
				curhps[i] = hp
			}
		}
	}
}

func getHP(name string) int {
	if charIsNpc(name) {
		for i := range npcs {
			if name == npcs[i].Key {
				return npcs[i].HP
			}
		}
	} else {
		for i := range chars {
			if chars[i].Key == name {
				return curhps[i]
			}
		}
	}

	return 0
}

func nextTurn() {
	if currentturn < len(outputar)-1 {
		currentturn++
	} else {
		currentturn = 1
	}
}

func prevTurn() {
	if currentturn > 1 {
		currentturn--
	}
}

func printStatus() {
	fmt.Println("Place: ", place)
	fmt.Println("Scene: ", scene)
	listNpcs()
	if initiativetxt != "" {
		fmt.Println(initiativetxt)
	}
}

func getCharWithTurn() Char {
	//for i := range outputar {
		parts := strings.Split(outputar[currentturn]," ")
		cname := strings.Join(parts[0:len(parts)-1], " ")
		fmt.Println("Current name: ", cname)
		if len(parts) >= 2 {
			fmt.Println("Ugh...", cname)

			for k := range npcs {
				if npcs[k].Name == cname {
					return npcs[k]
				}
			}

			for k := range chars {
				if chars[k].Name == cname && chars[k].InParty {
					return chars[k]
				}
			}

			fmt.Println("Failed to find npc or party char with turn...")


		}
		fmt.Println("Failed to lookup anything with ", outputar[currentturn])
	//}

	return npcs[0]

}

func getRandomNpc() Char {
	if len(npcs) == 1 {
		return npcs[0]
	}

	dres := getDiceResults(fmt.Sprintf("1d%d",len(npcs)))
	dres = strings.TrimRight(dres, " \n")
	ran, _ := strconv.Atoi(dres)
	return npcs[ran-1]
}

func getRandomPartyChar() Char {
	dres := getDiceResults(fmt.Sprintf("1d%d",len(players)))
	dres = strings.TrimRight(dres, " \n")
	ran, _ := strconv.Atoi(dres)


	for i := range chars {
		if chars[i].Playername == players[ran-1] {
			return chars[i]
		}
	}


	return Char{}
}

func loopForDMInput() {
	consolereader := bufio.NewReader(os.Stdin)
	for {

		input, _ := consolereader.ReadString('\n')
		cmd := parseInput(input)
		msg := ""
		//fmt.Println("Running command: ", cmd.Name)
		if (cmd.Name == "roll" || cmd.Name == "r") && len(cmd.Args) >= 1 {
			diceresults := getDiceResults(cmd.Args[0])
			if len(cmd.Args) == 2 {
				diceresults2 := getDiceResults(cmd.Args[0])
				msg = fmt.Sprintf("Roll 1 %s = %s", cmd.Args[0], diceresults)
				msg = msg + "<br>" + fmt.Sprintf("Roll 1 %s = %s", cmd.Args[0], diceresults2)
			} else {
				msg = fmt.Sprintf("Roll %s = %s", cmd.Args[0], diceresults)
			}
		} else if cmd.Name == "c" {
			msg = " "
		} else if cmd.Name == "ant" {
			cchar := getCharWithTurn()
			hp := getHP(cchar.Key)
			if cchar.InParty && hp > 0 {
				randomNpc := getRandomNpc()
				if randomNpc.HP <= 0 {
					fmt.Println("Target npc is dead!", randomNpc.HP, randomNpc.Key)
				} else {
					msg = attack(cchar, 0, randomNpc, "")
					nextTurn()
				}
			} else if cchar.InParty && hp < 0 {
				fmt.Println("Source is dead!")
				msg = cchar.Name + " is dead."
				nextTurn()
			} else if cchar.HP >= 0 && charIsNpc(cchar.Key) {
				randomChar := getRandomPartyChar()
				thp := getHP(randomChar.Key)
				if thp <= 0 {
					fmt.Println("Target player is dead!", thp, randomChar.Name)
				} else {
					msg = attack(cchar, 0, randomChar, "")
					nextTurn()
				}
			} else {
				fmt.Println("Source is dead!")
				msg = cchar.Name + " is dead."
				nextTurn()
			}
		} else if cmd.Name == "nt" {
			nextTurn()
			msg = " "
		} else if cmd.Name == "pt" {
			prevTurn()
			msg = " "
		} else if cmd.Name == "stat"  || cmd.Name == "status" {
			printStatus()
		} else if cmd.Name == "blog" {
			msg = battlelog
		} else if (cmd.Name == "rollq" || cmd.Name == "rq") && len(cmd.Args) >= 1 {
			diceresults := getDiceResults(cmd.Args[0])
			fmt.Printf("Roll %s = %s", cmd.Args[0], diceresults)
		} else if cmd.Name == "msg" {
			msg = fmt.Sprintf("%s", cmd.RawArgs)
		} else if cmd.Name == "drop" {
			dropNpc(cmd.Args[0])
			msg = " "
		} else if cmd.Name == "scene" || cmd.Name == "s" {
			initNpcs()
			if len(cmd.Args) == 0 {
				scene = ""
			} else {
				scene = cmd.Args[0]
				//fmt.Println(cmd.Args[0])
				sc, _ := getScene(cmd.Args[0])
				for i := range sc.Chars {
					dropNpc(sc.Chars[i])
				}
			}
			msg = " "
		} else if len(cmd.Args) >= 1 && (cmd.Name == "place" || cmd.Name == "p") {
			initNpcs()
			scene = ""
			place = cmd.Args[0]
			pl := getPlace(place)
			for i := range pl.Autodrop {
				dropNpc(pl.Autodrop[i])
			}
			msg = " "
		} else if cmd.Name == "vo" && len(cmd.Args) >= 1 {
			msg = viewObj(cmd.Args[0])
		} else if cmd.Name == "v" && len(cmd.Args) >= 1 {
			if charIsNpc(cmd.Args[0]) {
				msg = viewChar(cmd.Args[0],false)
			} else {
				msg = viewChar(cmd.Args[0],true)
			}
		} else if cmd.Name == "sethp" {
			aint, _ := strconv.Atoi(cmd.Args[1])
			setHP(cmd.Args[0], aint)
			msg = " "
		} else if cmd.Name == "subhp" && len(cmd.Args) >= 2 {
			aint, _ := strconv.Atoi(cmd.Args[1])
			hp := getHP(cmd.Args[0])
			setHP(cmd.Args[0], hp-aint)
			msg = " "
		} else if cmd.Name == "addhp" && len(cmd.Args) >= 2 {
			aint, _ := strconv.Atoi(cmd.Args[1])
			hp := getHP(cmd.Args[0])
			setHP(cmd.Args[0], hp+aint)
			msg = " "
		} else if cmd.Name == "t" {
			if NoText {
				NoText = false
			} else {
				NoText = true
			}
			msg = " "
		} else if cmd.Name == "sp" {
			if ShowParty {
				ShowParty = false
			} else {
				ShowParty = true
			}
			msg = " "
		} else if cmd.Name == "snp" {
			if ShowNpcs {
				ShowNpcs = false
			} else {
				ShowNpcs = true
			}
			msg = " "
		} else if cmd.Name == "smugs" {
			if ShowMugs {
				ShowMugs = false
			} else {
				ShowMugs = true
			}
			msg = " "
		} else if cmd.Name == "ls" {
			if len(cmd.Args) >= 1 && cmd.Args[0] == "places" {
				listPlaces()
			} else if len(cmd.Args) >= 1 && cmd.Args[0] == "chars" {
				listChars()
			} else if len(cmd.Args) >= 1 && cmd.Args[0] == "scenes" {
				listScenes()
			} else if len(cmd.Args) >= 1 && cmd.Args[0] == "objs" {
				listObjs()
			} else {
				listNpcs()
			}
		} else if cmd.Name == "clearnpcs" {
			initNpcs()
			msg = " "
		} else if cmd.Name == "combat" {
			initiativetxt = rollInitiatives(cmd.RawArgs)
			msg = " "
		} else if cmd.Name == "reset" {
			initialState()
			msg = " "
		} else if cmd.Name == "endcombat" {
			initiativetxt = ""
			outputar = make([]string, 0)
			battlelog = ""
			msg = " "
		} else if cmd.Name == "att" && len(cmd.Args) > 1 && strings.Contains(cmd.Args[0], ".") {
			a := strings.Split(cmd.Args[0], ".")
			atti, _ := strconv.Atoi(a[1])
			adv := ""
			if len(cmd.Args) == 3 {
				adv = cmd.Args[2]
			}
			char1 := getChar(a[0])
			char2 := getChar(cmd.Args[1])

			if char1.Name != "" && char2.Name != "" {
				msg = attack(char1, atti, char2, adv)
			} else if (char1.Name == "") {
				fmt.Println("Failed to load ", a[0])
			} else if (char2.Name == "") {
				fmt.Println("Failed to load ", cmd.Args[1])
			}
		} else if cmd.Name == "re" || cmd.Name == "reload" {
			fmt.Println("Reload Configuration")
			initPlaces()
			initChars(false)
			initObjects()
			initScenes()
			syncNpcs()
			msg = " "
		}

		fmt.Printf("> ")
		if msg != "" {
			lastoutput = renderContent(msg, cmd)
		}
		h.broadcast <- []byte(lastoutput)
	}
}

func loadPlayerChar(playername string) Char {
	filename := fmt.Sprintf("assets/players/%s.json",playername)
	file, err := os.Open(filename)

	if err != nil {
		panic(fmt.Sprintf("Could not open %s",filename))
	}

	filebytes := ReadFileContents(file)

	char := Char{}
	err = json.Unmarshal(filebytes, &char)
	if err != nil {
		fmt.Println("Failed to read player file: ", err)
		panic(err)
	}

	return char

}

func updatePlayerChar(req *http.Request) {
	char := Char{}

	char.Name = req.Form["charname"][0]
	char.Class = req.Form["class"][0]
	char.Race = req.Form["race"][0]
	char.Level,_ = strconv.Atoi(req.Form["level"][0])
	char.Alignment = req.Form["alignment"][0]
	abil := Abilities{}
	json.Unmarshal([]byte(req.Form["abilities"][0]), &abil)
	char.Abilities = abil
	//char.Abilities = req.Form["abilities"]
	char.HP,_ = strconv.Atoi(req.Form["hp"][0])
	char.AC,_ = strconv.Atoi(req.Form["ac"][0])
	char.Initiative,_ = strconv.Atoi(req.Form["initiative"][0])
	//char.Attacks,_ = req.Form["attacks"]
	atts := make([]Attack,0)
	json.Unmarshal([]byte(req.Form["attacks"][0]), &atts)
	char.Attacks = atts
	//char.Inventory,_ = req.Form["inventory"]
	inv := make([]string,0)
	//fmt.Println(inv)
	json.Unmarshal([]byte(req.Form["inventory"][0]), &inv)
	char.Inventory = inv
	char.Background = req.Form["background"][0]
	char.Image = req.Form["image"][0]
	char.Inspiration,_ = strconv.Atoi(req.Form["inspiration"][0])
	char.ProfBonus,_ = strconv.Atoi(req.Form["profbonus"][0])
	char.PassPerception,_ = strconv.Atoi(req.Form["passperception"][0])
	char.HitDice = req.Form["hitdice"][0]
	char.Speed,_ = strconv.Atoi(req.Form["speed"][0])
	char.Skills = req.Form["skills"][0]
	char.MiscProfLanguages = req.Form["misclang"][0]
	char.PersonalityTraits = req.Form["personalitytraits"][0]
	char.Ideals = req.Form["ideals"][0]
	char.Bonds = req.Form["bonds"][0]
	char.Flaws = req.Form["flaws"][0]
	char.FeaturesTraits = req.Form["featurestraits"][0]
	char.Treasure = req.Form["treasure"][0]

	char.SpellL0 = req.Form["spelll0"][0]
	char.SpellL1 = req.Form["spelll1"][0]
	char.SpellL2 = req.Form["spelll2"][0]
	char.SpellL3 = req.Form["spelll3"][0]
	char.SpellL4 = req.Form["spelll4"][0]
	char.SpellL5 = req.Form["spelll5"][0]
	char.SpellL6 = req.Form["spelll6"][0]
	char.SpellL7 = req.Form["spelll7"][0]

	if req.Form["inparty"][0] == "true" {
		char.InParty = true
	} else {
		char.InParty = false
	}
	char.Playername = req.Form["playername"][0]


	//fmt.Println(char)
	charbytes,_ := json.Marshal(char)
	//fmt.Println(string(bytes))

	buf := bytes.NewBuffer(make([]byte,0))
	json.Indent(buf,charbytes,"","  ")
	//fmt.Println(string(buf.Bytes()))
	fmt.Println(req.Form["playername"][0], "updated!")

	updatePlayerFile(req.Form["playername"][0],buf.Bytes())

}

func updatePlayerFile(charname string, data []byte) {


	if charname == "" {
		fmt.Println("Error empty playername passed to updatePlayerFile()\n")
	}
	filename := fmt.Sprintf("assets/players/%s.json",charname)

	err := ioutil.WriteFile(filename,data,0755)

	if err != nil {
		panic(fmt.Sprintf("Could not write to %s", filename, " ", err))
	}
	initPlaces()
	initChars(false)
	initObjects()
	initScenes()
	syncNpcs()
	lastoutput = renderContent("", &Command{})
}

func editPlayerChar(c http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	playercookie,_ := req.Cookie("playername")
	fmt.Println("Player cookie: ", playercookie.Value)
	if len(req.Form["update"]) != 0 {
		// update player data
		updatePlayerChar(req)
		http.Redirect(c,req,"/",302)
		//mainData := MainData{Host: req.Host, Content: "Successfully updated the char!<br><a href=\"/\">Return to main view</a><br>"}
		//homeTempl.Execute(c, mainData)
	} else if playercookie.Value != "" {
		// send form
		//mainData := MainData{Content: editPlayerForm(req)}
		char := loadPlayerChar(strings.ToLower(playercookie.Value))
		fmt.Println("Loaded ", char.Name)
		char.JsonIfy()
		editPlayerTempl.Execute(c, char)
	}
}

func webViewChar(c http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	playercookie,_ := req.Cookie("playername")
	if playercookie != nil {
		fmt.Println("Player cookie: ", playercookie.Value)
	}

	if len(req.Form["name"]) == 0 {
		mainData := MainData{Host: req.Host, Content: lastoutput}
		homeTempl.Execute(c, mainData)
	} else {
		char := getChar(req.Form["name"][0])
		// does the same thing now but may provide permissions to view in the future
		if playercookie != nil && char.Playername == playercookie.Value {
			mainData := MainData{Host: req.Host, Content: viewChar(req.Form["name"][0],true)}
			homeTempl.Execute(c, mainData)
		} else {
			mainData := MainData{Host: req.Host, Content: viewChar(req.Form["name"][0],true)}
			homeTempl.Execute(c, mainData)
		}
	}
}

var chttp = http.NewServeMux()

func initialState() {
	players = strings.Split(charlist,",")
	curhps = make(map[int]int)
	initPlaces()
	initObjects()
	initScenes()
	initChars(true)
	initNpcs()
	NoText = false
	ShowParty = true
	ShowMugs = true
	ShowNpcs = true
	place = "void"
	initiativetxt = ""
	outputar = make([]string, 0)
	battlelog = ""
	scene = ""
}

func main() {
	flag.StringVar(&charlist, "chars", "", "Character list separate by commas.")
	flag.Parse()

	homeTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "home.html")))
	editPlayerTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "editplayer.html")))
	playerIdTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "playerid.html")))

	initialState()

	lastoutput = renderContent("", &Command{})

	loggedin = make([]string,0)
	go h.run()
	go loopForDMInput()

	chttp.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
