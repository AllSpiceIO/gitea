// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"net/http"
	"testing"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/unknwon/i18n"
)

func createNewRelease(t *testing.T, session *TestSession, repoURL, tag, title string, preRelease, draft bool) {
	req := NewRequest(t, "GET", repoURL+"/releases/new")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	link, exists := htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")

	postData := map[string]string{
		"_csrf":      htmlDoc.GetCSRF(),
		"tag_name":   tag,
		"tag_target": "master",
		"title":      title,
		"content":    "",
	}
	if preRelease {
		postData["prerelease"] = "on"
	}
	if draft {
		postData["draft"] = "Save Draft"
	}
	req = NewRequestWithValues(t, "POST", link, postData)

	resp = session.MakeRequest(t, req, http.StatusFound)

	test.RedirectURL(resp) // check that redirect URL exists
}

func checkLatestReleaseAndCount(t *testing.T, session *TestSession, repoURL, version, label string, count int) {
	req := NewRequest(t, "GET", repoURL+"/releases")
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	labelText := htmlDoc.doc.Find("#release-list > li .meta .label").First().Text()
	assert.EqualValues(t, label, labelText)
	titleText := htmlDoc.doc.Find("#release-list > li .detail h4 a").First().Text()
	assert.EqualValues(t, version, titleText)

	releaseList := htmlDoc.doc.Find("#release-list > li")
	assert.EqualValues(t, count, releaseList.Length())
}

func TestViewReleases(t *testing.T) {
	defer prepareTestEnv(t)()

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/user2/repo1/releases")
	rsp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, rsp.Body)

	// Test compare dropdowns tags for each "release" (Draft Release and v1.1)
	compareElements := htmlDoc.Find(".item.choose.reference")
	tagNames := make([]string, 0, 5)
	compareUrls := make([]string, 0, 5)
	// Iterate each compare dropdown
	compareElements.Each(func(i int, s *goquery.Selection) {
		tag := s.Find(".item.tag")
		tagNames = append(tagNames, tag.Text())
		dataUrl, _ := tag.Attr("data-url")
		compareUrls = append(compareUrls, dataUrl)
	})

	// Text for both selectors should be "v1.1" (since it is the only tag)
	assert.EqualValues(t, []string{"v1.1", "v1.1"}, tagNames)
	// Compare for first selector should be "v1.1" against "master" since it is a draft release
	// Compare for second selector should be "v1.1" since "v1.1" is the only tag
	assert.EqualValues(t, []string{"/user2/repo1/compare/v1.1...master", "/user2/repo1/compare/v1.1...v1.1"}, compareUrls)
}

func TestViewReleasesNoLogin(t *testing.T) {
	defer prepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/releases")
	MakeRequest(t, req, http.StatusOK)
}

func TestCreateRelease(t *testing.T) {
	defer prepareTestEnv(t)()

	session := loginUser(t, "user2")
	createNewRelease(t, session, "/user2/repo1", "v0.0.1", "v0.0.1", false, false)

	checkLatestReleaseAndCount(t, session, "/user2/repo1", "v0.0.1", i18n.Tr("en", "repo.release.stable"), 3)

	req := NewRequest(t, "GET", "/user2/repo1/releases")
	rsp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, rsp.Body)

	// Test compare dropdowns tags for each "release" (New "v0.0.1" release, Draft Release, and "v1.1" release)
	compareElements := htmlDoc.Find(".item.choose.reference")
	tagNames := make([][]string, 0, 5)
	compareUrls := make([][]string, 0, 5)
	// Iterate each compare dropdown
	compareElements.Each(func(i int, s *goquery.Selection) {
		tags := s.Find(".item.tag")
		tempTagNames := make([]string, 0, 5)
		tempCompareUrls := make([]string, 0, 5)
		// Iterate each tag ref for each compare dropdown
		tags.Each(func(ii int, ss *goquery.Selection) {
			tempTagNames = append(tempTagNames, ss.Text())
			dataUrl, _ := ss.Attr("data-url")
			tempCompareUrls = append(tempCompareUrls, dataUrl)
		})
		tagNames = append(tagNames, tempTagNames)
		compareUrls = append(compareUrls, tempCompareUrls)
	})

	assert.EqualValues(t, [][]string{{"v0.0.1", "v1.1"}, {"v0.0.1", "v1.1"}, {"v0.0.1", "v1.1"}}, tagNames)
	assert.EqualValues(t, [][]string{
		{"/user2/repo1/compare/v0.0.1...v0.0.1", "/user2/repo1/compare/v1.1...v0.0.1"},
		{"/user2/repo1/compare/v0.0.1...master", "/user2/repo1/compare/v1.1...master"},
		{"/user2/repo1/compare/v0.0.1...v1.1", "/user2/repo1/compare/v1.1...v1.1"}}, compareUrls)
}

func TestCreateReleasePreRelease(t *testing.T) {
	defer prepareTestEnv(t)()

	session := loginUser(t, "user2")
	createNewRelease(t, session, "/user2/repo1", "v0.0.1", "v0.0.1", true, false)

	checkLatestReleaseAndCount(t, session, "/user2/repo1", "v0.0.1", i18n.Tr("en", "repo.release.prerelease"), 3)
}

func TestCreateReleaseDraft(t *testing.T) {
	defer prepareTestEnv(t)()

	session := loginUser(t, "user2")
	createNewRelease(t, session, "/user2/repo1", "v0.0.1", "v0.0.1", false, true)

	checkLatestReleaseAndCount(t, session, "/user2/repo1", "v0.0.1", i18n.Tr("en", "repo.release.draft"), 3)
}

func TestCreateReleasePaging(t *testing.T) {
	defer prepareTestEnv(t)()

	oldAPIDefaultNum := setting.API.DefaultPagingNum
	defer func() {
		setting.API.DefaultPagingNum = oldAPIDefaultNum
	}()
	setting.API.DefaultPagingNum = 10

	session := loginUser(t, "user2")
	// Create enaugh releases to have paging
	for i := 0; i < 12; i++ {
		version := fmt.Sprintf("v0.0.%d", i)
		createNewRelease(t, session, "/user2/repo1", version, version, false, false)
	}
	createNewRelease(t, session, "/user2/repo1", "v0.0.12", "v0.0.12", false, true)

	checkLatestReleaseAndCount(t, session, "/user2/repo1", "v0.0.12", i18n.Tr("en", "repo.release.draft"), 10)

	// Check that user4 does not see draft and still see 10 latest releases
	session2 := loginUser(t, "user4")
	checkLatestReleaseAndCount(t, session2, "/user2/repo1", "v0.0.11", i18n.Tr("en", "repo.release.stable"), 10)
}

func TestViewReleaseListNoLogin(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)

	link := repo.Link() + "/releases"

	req := NewRequest(t, "GET", link)
	rsp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, rsp.Body)
	releases := htmlDoc.Find("#release-list li.ui.grid")
	assert.Equal(t, 1, releases.Length())

	links := make([]string, 0, 5)
	releases.Each(func(i int, s *goquery.Selection) {
		link, exist := s.Find(".release-list-title a").Attr("href")
		if !exist {
			return
		}
		links = append(links, link)
	})

	assert.EqualValues(t, []string{"/user2/repo1/releases/tag/v1.1"}, links)
}

func TestViewReleaseListLogin(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)

	link := repo.Link() + "/releases"

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", link)
	rsp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, rsp.Body)
	releases := htmlDoc.Find("#release-list li.ui.grid")
	assert.Equal(t, 2, releases.Length())

	links := make([]string, 0, 5)
	releases.Each(func(i int, s *goquery.Selection) {
		link, exist := s.Find(".release-list-title a").Attr("href")
		if !exist {
			return
		}
		links = append(links, link)
	})

	assert.EqualValues(t, []string{"/user2/repo1/releases/tag/draft-release",
		"/user2/repo1/releases/tag/v1.1"}, links)
}

func TestViewTagsList(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)

	link := repo.Link() + "/tags"

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", link)
	rsp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, rsp.Body)
	tags := htmlDoc.Find(".tag-list tr")
	assert.Equal(t, 2, tags.Length())

	tagNames := make([]string, 0, 5)
	tags.Each(func(i int, s *goquery.Selection) {
		tagNames = append(tagNames, s.Find(".tag a.df.ac").Text())
	})

	assert.EqualValues(t, []string{"delete-tag", "v1.1"}, tagNames)
}
