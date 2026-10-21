package parse

import (
	"io"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Gallery はギャラリー scv_tgallery/open/{ギャラリーID}。本文は言語によらず同じ(clone-spec A 6)。
type Gallery struct {
	Name   string  `json:"name"`
	Photos []Photo `json:"photos"`
}

// Photo は写真 1 枚。URL は参照サイトの HTML にある生の空白などをエンコードした絶対 URL。
type Photo struct {
	ThumbURL string `json:"thumb_url"`
	FullURL  string `json:"full_url"`
}

// ParseGallery はギャラリーを読む。写真は #photo の a(原寸)と中の img(サムネイル)の組。
func ParseGallery(r io.Reader, lang site.Lang) (*Gallery, error) {
	if err := checkLang(lang); err != nil {
		return nil, err
	}
	doc, err := load(r)
	if err != nil {
		return nil, err
	}
	sv, err := scoreView(doc)
	if err != nil {
		return nil, err
	}
	out := &Gallery{Name: headingText(sv.Find("span.SCV_MIDA_M").First()), Photos: []Photo{}}
	var photoErr error
	sv.Find("#photo a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		full, err := site.AbsURL(a.AttrOr("href", ""))
		if err != nil {
			photoErr = unexpected("photo url: %v", err)
			return false
		}
		thumb, err := site.AbsURL(a.Find("img").AttrOr("src", ""))
		if err != nil {
			photoErr = unexpected("thumbnail url: %v", err)
			return false
		}
		out.Photos = append(out.Photos, Photo{ThumbURL: thumb, FullURL: full})
		return true
	})
	if photoErr != nil {
		return nil, photoErr
	}
	return out, nil
}

// MergeGalleryEN は英語版が日本語版と同じ内容かを確かめ、日本語版を返す。ギャラリーに
// 言語で変わる文字列はない(clone-spec A 6)。
func MergeGalleryEN(jp, en *Gallery) (*Gallery, error) {
	if jp.Name != en.Name || len(jp.Photos) != len(en.Photos) {
		return nil, mismatch("gallery %q differs between jp and en", jp.Name)
	}
	out := *jp
	return &out, nil
}
