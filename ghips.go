package main

import (
  "fmt"
  "github.com/google/go-github/github"
  "golang.org/x/oauth2"
  "log"
  "os"
  "strings"
  "time"
)

var (
  personalAccessToken string
  issuesCollection    allIssues
  org                 string
)

type TokenSource struct {
  AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
  token := &oauth2.Token{
    AccessToken: t.AccessToken,
  }
  return token, nil
}

type allIssues struct {
  issues                                                                              github.IssuesSearchResult
  users                                                                               []github.User
  issues_new_public, issues_2m_public, issues_6m_public, issues_1y_public             []github.Issue
  issues_new_orgmember, issues_2m_orgmember, issues_6m_orgmember, issues_1y_orgmember []github.Issue
  pr_new_public, pr_2m_public, pr_6m_public, pr_1y_public                             []github.Issue
  pr_new_orgmember, pr_2m_orgmember, pr_6m_orgmember, pr_1y_orgmember                 []github.Issue
}

func main() {
  org = os.Getenv("GHIPS_ORG")
  personalAccessToken = os.Getenv("GITHUB_ACCESS_TOKEN")

  fmt.Println(org)
  if len(personalAccessToken) == 0 {
    log.Fatal("Before you can use this you must set the GITHUB_ACCESS_TOKEN environment variable.")
  }

  tokenSource := &TokenSource{
    AccessToken: personalAccessToken,
  }
  oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

  client := github.NewClient(oauthClient)
  twomonthsago := time.Now().Add(time.Hour * 24 * 30 * 2 * -1)
  sixmonthsago := time.Now().Add(time.Hour * 24 * 30 * 6 * -1)
  oneyearago := time.Now().Add(time.Hour * 24 * 365 * -1)

  err := populateUsers(org, client)
  if err != nil {
    fmt.Println("Error getting users for " + org)
  }

  err = populateIssues(org, client)
  if err != nil {
    fmt.Println("Error getting issues for " + org)
  }

  totalPR := 0
  totalIssues := 0
  for _, issue := range issuesCollection.issues.Issues {
    if issue.PullRequestLinks == nil {
      totalIssues++
      if issue.UpdatedAt.After(twomonthsago) {
        populateIssueGroup(&issuesCollection.issues_new_orgmember, &issuesCollection.issues_new_public, issue.User, issue)
      } else if issue.UpdatedAt.After(sixmonthsago) {
        populateIssueGroup(&issuesCollection.issues_2m_orgmember, &issuesCollection.issues_2m_public, issue.User, issue)
      } else if issue.UpdatedAt.After(oneyearago) {
        populateIssueGroup(&issuesCollection.issues_6m_orgmember, &issuesCollection.issues_6m_public, issue.User, issue)
      } else {
        populateIssueGroup(&issuesCollection.issues_1y_orgmember, &issuesCollection.issues_1y_public, issue.User, issue)
      }
    } else {
      totalPR++
      if issue.UpdatedAt.After(twomonthsago) {
        populateIssueGroup(&issuesCollection.pr_new_orgmember, &issuesCollection.pr_new_public, issue.User, issue)
      } else if issue.UpdatedAt.After(sixmonthsago) {
        populateIssueGroup(&issuesCollection.pr_2m_orgmember, &issuesCollection.pr_2m_public, issue.User, issue)
      } else if issue.UpdatedAt.After(oneyearago) {
        populateIssueGroup(&issuesCollection.pr_6m_orgmember, &issuesCollection.pr_6m_public, issue.User, issue)
      } else {
        populateIssueGroup(&issuesCollection.pr_1y_orgmember, &issuesCollection.pr_1y_public, issue.User, issue)
      }
    }
  }

  fmt.Printf("\nSummary\n")
  fmt.Printf("\nPull Requests - %d\n", totalPR)
  fmt.Printf("\n  Employee Pull Requests")
  fmt.Printf("\n    New: \t\t%d\n    2-6 months old:\t%d\n    6-12 months old:\t%d\n    Older that 1 year:\t%d", len(issuesCollection.pr_new_orgmember), len(issuesCollection.pr_2m_orgmember), len(issuesCollection.pr_6m_orgmember), len(issuesCollection.pr_1y_orgmember))
  fmt.Printf("\n\n  Public Pull Requests")
  fmt.Printf("\n    New: \t\t%d\n    2-6 months old:\t%d\n    6-12 months old:\t%d\n    Older that 1 year:\t%d\n", len(issuesCollection.pr_new_public), len(issuesCollection.pr_2m_public), len(issuesCollection.pr_6m_public), len(issuesCollection.pr_1y_public))

  fmt.Printf("\nIssues - %d\n", totalIssues)
  fmt.Printf("\n  Employee Issues")
  fmt.Printf("\n    New: \t\t%d\n    2-6 months old:\t%d\n    6-12 months old:\t%d\n    Older that 1 year:\t%d", len(issuesCollection.issues_new_orgmember), len(issuesCollection.issues_2m_orgmember), len(issuesCollection.issues_6m_orgmember), len(issuesCollection.issues_1y_orgmember))
  fmt.Printf("\n\n  Public Issues")
  fmt.Printf("\n    New: \t\t%d\n    2-6 months old:\t%d\n    6-12 months old:\t%d\n    Older that 1 year:\t%d\n", len(issuesCollection.issues_new_public), len(issuesCollection.issues_2m_public), len(issuesCollection.issues_6m_public), len(issuesCollection.issues_1y_public))

  fmt.Printf("\n\nPublic Details - Pull Requests\n")
  printIssues(issuesCollection.pr_1y_public, "1+ year old pull requests")
  printIssues(issuesCollection.pr_6m_public, "6-12 month old pull requests")
  printIssues(issuesCollection.pr_2m_public, "2-6 month old pull requests")
  printIssues(issuesCollection.pr_new_public, "New Pull Requests")

  fmt.Printf("\n\nPublic Details - Issues\n")
  printIssues(issuesCollection.issues_1y_public, "1+ year old issues")
  printIssues(issuesCollection.issues_6m_public, "6-12 month old issues")
  printIssues(issuesCollection.issues_2m_public, "2-6 month old issues")
  printIssues(issuesCollection.issues_new_public, "New Issues")
}

func printIssues(issues []github.Issue, title string) {
  fmt.Printf("  %s\n", title)
  for _, issue := range issues {
    var title string
    if len(*issue.Title) > 60 {
      title = (*issue.Title)[:59] + "..."
    } else {
      title = *issue.Title
    }
    getRepoName(issue)

    fmt.Printf("  %-23s %-19s %-2s(%5d) %-62s (%s)\n", getRepoName(issue), *issue.User.Login, attentionStatus(issue), *issue.Number, title, issue.UpdatedAt.Format("02-01-06"))
  }
  fmt.Printf("\n")
}

func attentionStatus(issue github.Issue) (attention_status string) {
  attention_status = ""
  if *issue.Comments == 0 && issue.CreatedAt.Before(time.Now().Add(time.Hour*24*2*-1)) {
    attention_status = "**"
  }

  return
}
func getRepoName(issue github.Issue) string {
  url := *issue.URL
  // fmt.Println(url)
  // fmt.Println(url[30+len(org):])
  // fmt.Println(url[:strings.LastIndex(url, "issues")])
  // fmt.Println(url[30+len(org)])
  repo := (url)[30+len(org) : strings.LastIndex(url, "issues")-1]
  return repo
}
func isUserAnOrgMember(thisuser github.User) bool {
  for _, user := range issuesCollection.users {
    if *thisuser.Login == *user.Login {
      return true
    }
  }
  return false
}

func populateIssueGroup(memberissuelist *[]github.Issue, publicissuelist *[]github.Issue, user *github.User, issue github.Issue) {
  if isUserAnOrgMember(*user) {
    *memberissuelist = append(*memberissuelist, issue)
  } else {
    *publicissuelist = append(*publicissuelist, issue)
  }
}

func populateUsers(org string, client *github.Client) (err error) {
  useropt := &github.ListMembersOptions{
    ListOptions: github.ListOptions{},
  }
  for {
    userSubset, resp, err := client.Organizations.ListMembers(org, useropt)
    fmt.Print(".")
    if err != nil {
      return err
    }
    issuesCollection.users = append(issuesCollection.users, userSubset...)
    if resp.NextPage == 0 {
      break
    }
    useropt.ListOptions.Page = resp.NextPage
  }
  return
}

func populateIssues(org string, client *github.Client) (err error) {
  issueopt := &github.SearchOptions{
    ListOptions: github.ListOptions{PerPage: 100},
  }

  for {
    issuesSubset, resp, err := client.Search.Issues("is:open is:public user:"+org, issueopt)
    fmt.Print("....")
    if err != nil {
      return err
    }
    issuesCollection.issues.Issues = append(issuesCollection.issues.Issues, issuesSubset.Issues...)
    if resp.NextPage == 0 {
      break
    }
    issueopt.ListOptions.Page = resp.NextPage
  }
  return
}
