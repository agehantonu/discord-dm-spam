package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

func clearConsole() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func printGradientA() {
	asciiA := []string{
		" #                          #         ##    ##   ###   ###   ###   ####  #  #",
		"                            #     #  #  #  #  #  #  #  #  #  #  #  #     #  # ",
		"##     ###          ###   ###    #   #  #  #  #  #  #  #  #  #  #  ###   #  #",
		" #    ##           #  #  #  #   #    ####  ####  ###   ###   #  #  #     #  #",
		" #      ##    ##    ##   #  #  #     #  #  #  #  # #   # #   #  #  #      ##",
		"###   ###     ##   #      ###        #  #  #  #  #  #  #  #  ###   ####   ##",
		"                    ###",
	}
	for i, line := range asciiA {
		r := int(float64(i) / float64(len(asciiA)-1) * 255)
		b := 255 - r
		fmt.Printf("\x1b[38;2;%d;0;%dm%s\x1b[0m\n", r, b, line)
	}
	fmt.Println("\n     dev discord - ag3q  ")
}

func getIcon(url string) string {
	if url == "" { return "" }
	resp, err := http.Get(url)
	if err != nil { return "" }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return fmt.Sprintf("data:%s;base64,%s", http.DetectContentType(b), base64.StdEncoding.EncodeToString(b))
}


func request(method, url, tk string, body interface{}) (*http.Response, []byte) {
	var b io.Reader
	if body != nil {
		j, _ := json.Marshal(body)
		b = bytes.NewBuffer(j)
	}
	req, _ := http.NewRequest(method, "https://discord.com/api/v9"+url, b)
	req.Header.Set("Authorization", tk)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return nil, nil }
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func main() {
	clearConsole()
	printGradientA()
	f, _ := os.Open("tokens.txt")
	var tokens []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if t := strings.TrimSpace(sc.Text()); t != "" { tokens = append(tokens, t) }
	}
	f.Close()

	fmt.Print("\n[1]作成 [2]退出 > ")
	var mode string
	fmt.Scanln(&mode)
	rd := bufio.NewReader(os.Stdin)

	if mode == "1" {
		fmt.Print("ID(カンマ区切り): ")
		uInput, _ := rd.ReadString('\n')
		uIDs := strings.Split(strings.TrimSpace(uInput), ",")
		fmt.Print("作成するグループ名: ")
		name, _ := rd.ReadString('\n')
		fmt.Print("画像URL: ")
		url, _ := rd.ReadString('\n')
		fmt.Print("送信する文: ")
		msg, _ := rd.ReadString('\n')
		fmt.Print("作成数: ")
		var count int
		fmt.Scanln(&count)

		icon := getIcon(strings.TrimSpace(url))
		var wg sync.WaitGroup
		for _, t := range tokens {
			wg.Add(1)
			go func(tk string) {
				defer wg.Done()
				runCreate(tk, uIDs, strings.TrimSpace(name), icon, strings.TrimSpace(msg), count)
			}(t)
		}
		wg.Wait()
	} else {
		fmt.Print("抜けるグループ名: ")
		target, _ := rd.ReadString('\n')
		fmt.Print("変える名前: ")
		newName, _ := rd.ReadString('\n')
		fmt.Print("画像URL: ")
		url, _ := rd.ReadString('\n')

		icon := getIcon(strings.TrimSpace(url))
		for _, t := range tokens {
			runExit(t, strings.TrimSpace(target), strings.TrimSpace(newName), icon)
		}
	}
}

func runCreate(tk string, uIDs []string, name, icon, msg string, total int) {
	q := make(chan string, total)
	go func() {
		for i := 0; i < total; i++ {
		L1:
			resp, data := request("POST", "/users/@me/channels", tk, map[string]interface{}{"recipients": []string{}})
			if resp == nil || resp.StatusCode != 200 {
				fmt.Println("[リミットレート]グループ作成失敗-10分待ちます")
				time.Sleep(10 * time.Minute)
				goto L1
			}
			var ch channel
			json.Unmarshal(data, &ch)
			
			request("PATCH", "/channels/"+ch.ID, tk, map[string]interface{}{"name": name, "icon": icon})
			fmt.Printf("✅ %s-%s-%s-作成成功\n", strings.Join(uIDs, ","), name, ch.ID)
			q <- ch.ID
			time.Sleep(100 * time.Millisecond)
		}
		close(q)
	}()

	var batch []string
	for gid := range q {
		batch = append(batch, gid)
		if len(batch) >= 10 || len(batch) == total {
			for _, id := range batch {
				for _, uid := range uIDs {
				L2:
					rAdd, _ := request("PUT", "/channels/"+id+"/recipients/"+uid, tk, nil)
					if rAdd == nil || rAdd.StatusCode > 299 {
						fmt.Println("[リミットレート]グループ追加失敗-2分待ちます")
						request("POST", "/channels/"+id+"/messages", tk, map[string]string{"content": msg})
						time.Sleep(2 * time.Minute)
						goto L2
					}
					fmt.Printf("✅ <%s>-追加成功-%s\n", uid, id)
					time.Sleep(500 * time.Millisecond)
				}
				request("POST", "/channels/"+id+"/messages", tk, map[string]string{"content": msg})
			}
			batch = nil
		}
	}
}

func runExit(tk, target, newName, icon string) {
	resp, data := request("GET", "/users/@me/channels", tk, nil)
	if resp == nil || resp.StatusCode != 200 { return }
	
	var chs []channel
	json.Unmarshal(data, &chs)

	for _, ch := range chs {
		if ch.Type == 3 && strings.Contains(ch.Name, target) { 
			res := []string{"<" + ch.ID + ">"}
			if r, _ := request("PATCH", "/channels/"+ch.ID, tk, map[string]interface{}{"name": newName}); r != nil && r.StatusCode == 200 {
				res = append(res, "名前変更成功")
			}
			if icon != "" {
				if r, _ := request("PATCH", "/channels/"+ch.ID, tk, map[string]interface{}{"icon": icon}); r != nil && r.StatusCode == 200 {
					res = append(res, "アイコン変更成功")
				}
			}
			time.Sleep(1 * time.Second)
			if r, _ := request("DELETE", "/channels/"+ch.ID, tk, nil); r != nil && r.StatusCode < 300 {
				res = append(res, "退出成功")
			}
			fmt.Printf("✅ %s\n", strings.Join(res, "-"))
			time.Sleep(300 * time.Millisecond)
		}
	}
}
