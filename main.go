package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/yukpiz/transfer-github/github"
	"github.com/yukpiz/transfer-github/http"
)

var (
	organizationName = flag.String("org", "", "github organization name")
	clientID         = flag.String("id", "", "github client id")
	clientSecret     = flag.String("secret", "", "github client secret")
	newOwnerName     = flag.String("new", "", "new owner github accout name")
	collaboUsers     = flag.String("users", "", "new collaborator users")
)

const (
	AuthorizeURL          = "https://github.com/login/oauth/authorize"
	AccessTokenURL        = "https://github.com/login/oauth/access_token"
	OrgReposURL           = "https://api.github.com/orgs/%s/repos"
	TransferReposURL      = "https://api.github.com/repos/%s/%s/transfer"
	InviteCollaboratorURL = "https://api.github.com/repos/%s/%s/collaborators/%s"
)

func main() {
	flag.Parse()

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=http://localhost&scope=repo", AuthorizeURL, *clientID)
	openbrowser(authURL)

	var cd string
	fmt.Printf("Please Access To: %s\n", authURL)
	fmt.Printf("Authorization Code> ")
	fmt.Scanf("%s", &cd)
	fmt.Printf("\n")

	res, err := http.PostJSON(
		AccessTokenURL,
		map[string]string{
			"Accept": "application/json",
		},
		map[string]string{
			"client_id":     *clientID,
			"client_secret": *clientSecret,
			"code":          cd,
		})
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var at github.AccessToken
	if err := json.Unmarshal(b, &at); err != nil {
		panic(err)
	}

	// 1. get organization repositories
	var repos []*github.Repository
	headers := genAuthenticationHeader(at.AccessToken)
	headers["Accept"] = "application/vnd.github.nebula-preview+json"
	pg := 1
	for {
		res, err := http.Get(
			fmt.Sprintf(OrgReposURL, *organizationName),
			headers,
			map[string]string{
				"type":     "all",
				"page":     strconv.Itoa(pg),
				"per_page": strconv.Itoa(100),
			},
		)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		var rs []*github.Repository
		if err := json.Unmarshal(b, &rs); err != nil {
			panic(err)
		}
		if rs != nil && len(rs) > 0 {
			repos = append(repos, rs...)
		} else {
			break
		}
		pg++
	}

	for _, repo := range repos {
		log.Println(repo.Name)
	}

	testrepo := "test-transfer-repo2"

	// TODO: transfer repository
	res, err = http.PostJSON(
		fmt.Sprintf(TransferReposURL, *organizationName, testrepo),
		genAuthenticationHeader(at.AccessToken),
		map[string]string{
			"new_owner": *newOwnerName,
		})
	if err != nil {
		panic(err)
	}
	fmt.Printf("transfer: %s => %d\n", testrepo, res.StatusCode)

	// TODO: invite cont
	users := strings.Split(*collaboUsers, ",")
	for _, user := range users {
		res, err = http.Put(
			fmt.Sprintf(InviteCollaboratorURL, *organizationName, testrepo, user),
			genAuthenticationHeader(at.AccessToken),
			map[string]string{
				"permission": "admin",
			},
		)
		fmt.Printf("collaborator: %s => %d\n", user, res.StatusCode)
	}
}

func genAuthenticationHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": genAuthentication(token),
	}
}

func genAuthentication(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}
