package core

// Real-page fixtures for task 21 (BUG-013). Captured 2026-09-27 from the live
// watch pages with docs/evidence/bug-013/capture_fixtures.py: the
// captions.playerCaptionsTracklistRenderer captionTracks and audioTrackId list,
// verbatim except the long signed "baseUrl" fields (stripped, as task 18 did),
// in the page's real track order (the `.4` id is NOT first in audioTracks).
//
// Expected values are confirmed by the yt-dlp runs in docs/evidence/bug-013/
// (yt-dlp "language": r8CppXSqVDU vi, cZSgL76ddDs de, vyIgAO8aCbA en; `.4`
// equalled it on 14/14 structured videos).

// r8CppXSqVDU
var (
	fxTracksVi = `[{"name":{"simpleText":"Tiếng Anh (Hoa Kỳ) (được tạo tự động)"},"vssId":"a.en-US","languageCode":"en-US","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Việt (được tạo tự động)"},"vssId":"a.vi","languageCode":"vi","kind":"asr","isTranslatable":true,"trackName":""}]`
	fxAudioVi  = []string{"vi.4", "en-US.10"}
)

// cZSgL76ddDs
var (
	fxTracksDe = `[{"name":{"simpleText":"Tiếng Anh (Hoa Kỳ) (được tạo tự động)"},"vssId":"a.en-US","languageCode":"en-US","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Đức (được tạo tự động)"},"vssId":"a.de","languageCode":"de","kind":"asr","isTranslatable":true,"trackName":""}]`
	fxAudioDe  = []string{"de-DE.4", "en-US.10"}
)

// vyIgAO8aCbA
var (
	fxTracksEn21 = `[{"name":{"simpleText":"Tiếng Ả Rập (được tạo tự động)"},"vssId":"a.ar","languageCode":"ar","kind":"asr","rtl":true,"isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Anh (được tạo tự động)"},"vssId":"a.en","languageCode":"en","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Ba Lan (được tạo tự động)"},"vssId":"a.pl","languageCode":"pl","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Bangla (được tạo tự động)"},"vssId":"a.bn","languageCode":"bn","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Bồ Đào Nha (Brazil) (được tạo tự động)"},"vssId":"a.pt-BR","languageCode":"pt-BR","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Do Thái (được tạo tự động)"},"vssId":"a.iw","languageCode":"iw","kind":"asr","rtl":true,"isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Đức (Đức) (được tạo tự động)"},"vssId":"a.de-DE","languageCode":"de-DE","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Hà Lan (Hà Lan) (được tạo tự động)"},"vssId":"a.nl-NL","languageCode":"nl-NL","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Hàn (được tạo tự động)"},"vssId":"a.ko","languageCode":"ko","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Hindi (được tạo tự động)"},"vssId":"a.hi","languageCode":"hi","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Indonesia (được tạo tự động)"},"vssId":"a.id","languageCode":"id","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Italy (được tạo tự động)"},"vssId":"a.it","languageCode":"it","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Malayalam (được tạo tự động)"},"vssId":"a.ml","languageCode":"ml","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Nga (được tạo tự động)"},"vssId":"a.ru","languageCode":"ru","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Nhật (được tạo tự động)"},"vssId":"a.ja","languageCode":"ja","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Pháp (Pháp) (được tạo tự động)"},"vssId":"a.fr-FR","languageCode":"fr-FR","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Punjab (được tạo tự động)"},"vssId":"a.pa","languageCode":"pa","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Tamil (được tạo tự động)"},"vssId":"a.ta","languageCode":"ta","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Tây Ban Nha (Hoa Kỳ) (được tạo tự động)"},"vssId":"a.es-US","languageCode":"es-US","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Telugu (được tạo tự động)"},"vssId":"a.te","languageCode":"te","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Tiếng Ukraina (được tạo tự động)"},"vssId":"a.uk","languageCode":"uk","kind":"asr","isTranslatable":true,"trackName":""}]`
	fxAudioEn21  = []string{"ta.10", "bn.10", "ar.10", "de-DE.10", "te.10", "iw.10", "ja.10", "ml.10", "ko.10", "nl-NL.10", "en-US.4", "pa.10", "es-US.10", "fr-FR.10", "hi.10", "id.10", "ru.10", "uk.10", "pt-BR.10", "pl.10", "it.10"}
)
