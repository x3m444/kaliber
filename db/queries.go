/*
   Copyright © 2019, 2020 M.Watermann, 10247 Berlin, Germany
                  All rights reserved
               EMail : <support@mwat.de>
*/

package db

//lint:file-ignore ST1005 - I prefer capitalisation
//lint:file-ignore ST1017 - I prefer Yoda conditions

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// `quBaseQuery` is the default query to get all document data.
	// By appending `WHERE` and `LIMIT` clauses the result-set can
	// get stinted (i.e. limited to a sub-set).
	quBaseQuery = `SELECT b.id,
b.title,
IFNULL((SELECT group_concat(a.name || "|" || a.id, ", ")
	FROM authors a
	JOIN books_authors_link bal ON(bal.author = a.id)
	WHERE (bal.book = b.id)
), "") authors,
IFNULL((SELECT group_concat(p.name || "|" || p.id)
	FROM publishers p
	JOIN books_publishers_link bpl ON(p.id = bpl.publisher)
	WHERE (bpl.book = b.id)
), "") publisher,
IFNULL((SELECT r.rating
	FROM ratings r
	WHERE r.id IN (
		SELECT brl.rating
		from books_ratings_link brl
		WHERE (brl.book = b.id)
	)
), 0) rating,
b.timestamp,
IFNULL((SELECT MAX(data.uncompressed_size)
	FROM data
	WHERE (data.book = b.id)
), 0) size,
IFNULL((SELECT group_concat(t.name || "|" || t.id, ", ")
	FROM tags t
	JOIN books_tags_link btl ON(btl.tag = t.id)
	WHERE (btl.book = b.id)
), "") tags,
IFNULL((SELECT c.text
	FROM comments c
	WHERE (c.book = b.id)
), "") comments,
IFNULL((SELECT group_concat(s.name || "|" || s.id, ", ")
	FROM series s
	JOIN books_series_link bsl ON(bsl.series = s.id)
	WHERE (bsl.book = b.id)
), "") series,
b.series_index,
b.sort AS title_sort,
b.author_sort,
IFNULL((SELECT group_concat(d.format || "|" || d.id, ", ")
	FROM data d
	WHERE (d.book = b.id)
), "") formats,
IFNULL((SELECT group_concat(l.lang_code || "|" || l.id, ", ")
	FROM books_languages_link bll
	JOIN languages l ON(bll.lang_code = l.id)
	WHERE (bll.book = b.id)
), "") languages,
b.isbn,
IFNULL((SELECT group_concat(i.type || "|" || i.id || "|" || i.val, ", ")
	FROM identifiers i
	WHERE (i.book = b.id)
), "") identifiers,
b.path,
b.lccn,
b.pubdate,
b.flags,
b.uuid,
b.has_cover
FROM books b `
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type (
	// A pipe separated value string
	tPSVstring = string
)

// `doQueryAll()` returns a list of documents with all available fields
// and an `error` in case of problems.
//
//	`aQuery` The SQL query to run.
func doQueryAll(aQuery string) (*TDocList, error) {
	rows, err := dbSqliteDB.Query(aQuery)
	if nil != err {
		return nil, err
	}
	defer rows.Close()

	result := NewDocList()
	for rows.Next() {
		var (
			authors, formats, identifiers, languages,
			publisher, series, tags tPSVstring
			noTime  time.Time
			visible bool
		)
		doc := NewDocument()
		_ = rows.Scan(&doc.ID, &doc.Title, &authors, &publisher,
			&doc.Rating, &doc.timestamp, &doc.Size, &tags,
			&doc.comments, &series, &doc.seriesindex,
			&doc.titleSort, &doc.authorSort, &formats, &languages,
			&doc.ISBN, &identifiers, &doc.path, &doc.lccn,
			&doc.pubdate, &doc.flags, &doc.uuid, &doc.hasCover)

		// check for (un)visible fields:
		if visible, _ = BookFieldVisible(`authors`); !visible {
			visible, _ = BookFieldVisible(`author_sort`)
		}
		if visible {
			doc.authors = prepAuthors(authors)
		}
		if visible, _ = BookFieldVisible(`comments`); !visible {
			doc.comments = ``
		}
		if visible, _ = BookFieldVisible(`formats`); visible {
			doc.formats = prepFormats(formats)
		}
		if visible, _ = BookFieldVisible(`identifiers`); visible {
			doc.identifiers = prepIdentifiers(identifiers)
		}
		if visible, _ = BookFieldVisible(`languages`); visible {
			doc.languages = prepLanguages(languages)
		}
		if visible, _ = BookFieldVisible(`#pages`); visible {
			doc.Pages = prepPages(doc.path)
		}
		if visible, _ = BookFieldVisible(`path`); !visible {
			doc.path = ``
		}
		if visible, _ = BookFieldVisible(`pubdate`); !visible {
			doc.pubdate = noTime
		}
		if visible, _ = BookFieldVisible(`publisher`); visible {
			doc.publisher = prepPublisher(publisher)
		}
		if visible, _ = BookFieldVisible(`rating`); !visible {
			doc.Rating = 0
		}
		if visible, _ = BookFieldVisible(`series`); visible {
			doc.series = prepSeries(series)
		}
		if visible, _ = BookFieldVisible(`tags`); visible {
			doc.tags = prepTags(tags)
		}
		if visible, _ = BookFieldVisible(`timestamp`); !visible {
			doc.timestamp = noTime
		}
		if visible, _ = BookFieldVisible(`title`); !visible {
			visible, _ = BookFieldVisible(`sort`)
		}
		if !visible {
			doc.Title = ``
		}
		if visible, _ = BookFieldVisible(`size`); !visible {
			doc.Size = 0
		}
		if visible, _ = BookFieldVisible(`uuid`); !visible {
			doc.uuid = ``
		}

		result.Add(doc)
	}

	return result, nil
} // doQueryAll()

// `escapeQuery()` returns a string with some characters escaped.
//
// see: https://github.com/golang/go/issues/18478#issuecomment-357285669
func escapeQuery(aSource string) string {
	sLen := len(aSource)
	if 0 == sLen {
		return ``
	}
	var (
		character byte
		escape    bool
		j         int
	)
	result := make([]byte, sLen<<1)
	for i := 0; i < sLen; i++ {
		character = aSource[i]
		switch character {
		case '\n', '\r', '\\', '"':
			// We do not escape the apostrophe since it can be
			// a legitimate part of the search term and we use
			// double quotes to enclose those terms.
			escape = true
		case '\032': // Ctrl-Z
			escape = true
			character = 'Z'
		default:
		}

		if escape {
			escape = false
			result[j] = '\\'
			result[j+1] = character
			j += 2
		} else {
			result[j] = character
			j++
		}
	}

	return string(result[0:j])
} // escapeQuery()

var (
	quHaving = map[string]string{
		`all`:       ``,
		`authors`:   `JOIN books_authors_link a ON(a.book = b.id) WHERE (a.author = %d) `,
		`format`:    `JOIN data d ON(b.id = d.book) JOIN data dd ON (d.format = dd.format) WHERE (dd.id = %d) `,
		`languages`: `JOIN books_languages_link l ON(l.book = b.id) WHERE (l.lang_code = %d) `,
		`publisher`: `JOIN books_publishers_link p ON(p.book = b.id) WHERE (p.publisher = %d) `,
		`series`:    `JOIN books_series_link s ON(s.book = b.id) WHERE (s.series = %d) `,
		`tags`:      `JOIN books_tags_link t ON(t.book = b.id) WHERE (t.tag = %d) `,
	}
)

// `having()` returns a string limiting the query to the given `aEntity`
// with `aID`.
func having(aEntity string, aID TID) string {
	if (0 == len(aEntity)) || (`all` == aEntity) || (0 == aID) {
		return ``
	}

	return fmt.Sprintf(quHaving[aEntity], aID)
} // having()

// `limit()` returns a LIMIT clause defined by `aStart` and `aLength`.
func limit(aStart, aLength uint) string {
	return `LIMIT ` + strconv.FormatInt(int64(aStart), 10) +
		`,` + strconv.FormatInt(int64(aLength), 10)
} // limit()

// `orderBy()` returns a ORDER_BY clause defined by `aOrder` and `aDesc`.
//
// The `aOrder` argument can be one of the following constants:
//
//	qoSortUnsorted      = TSortType(iota)
//	qoSortByAuthor
//	qoSortByLanguage
//	qoSortByPublisher
//	qoSortByRating
//	qoSortBySeries
//	qoSortBySize
//	qoSortByTags
//	qoSortByTime
//	qoSortByTitle
//
//	`aDescending` If `true` the query result is sorted in DESCending order.
func orderBy(aOrder TSortType, aDescending bool) string {
	desc := `` // ` ASC ` is default
	if aDescending {
		desc = ` DESC`
	}
	var result string
	switch aOrder { // constants defined in `queryoptions.go`
	case qoSortByAcquisition:
		result = `b.timestamp` + desc + `, b.pubdate` + desc + `, b.author_sort`
	case qoSortByAuthor:
		result = `b.author_sort` + desc + `, b.pubdate`
	case qoSortByLanguage:
		result = `languages` + desc + `, b.author_sort` + desc + `, b.sort`
	case qoSortByPublisher:
		result = `publisher` + desc + `, b.author_sort` + desc + `, b.sort`
	case qoSortByRating:
		result = `rating` + desc + `, b.author_sort` + desc + `, b.sort`
	case qoSortBySeries:
		result = `series` + desc + `, b.series_index` + desc + `, b.sort`
	case qoSortBySize:
		result = `size` + desc + `, b.author_sort`
	case qoSortByTags:
		result = `tags` + desc + `, b.author_sort`
	case qoSortByTime:
		result = `b.pubdate` + desc + `, b.timestamp` + desc + `, b.author_sort`
	case qoSortByTitle:
		result = `b.sort` + desc + `, b.author_sort`
	default:
		return ``
	}

	return ` ORDER BY ` + result + desc + ` `
} // orderBy()

// `prepAuthors()` returns a sorted list of document authors.
//
//	`aAuthor` The document's author(s).
func prepAuthors(aAuthor tPSVstring) *tAuthorList {
	list := strings.Split(aAuthor, `, `)
	result := make(tAuthorList, 0, len(list))
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		result = append(result, TEntity{
			ID:   num,
			Name: a[0],
		})
	}
	if 1 < len(result) {
		sort.Slice(result, func(i, j int) bool {
			return (result[i].Name < result[j].Name) // descending
		})
	}

	return &result
} // prepAuthors()

// `prepFormats()` returns a sorted list of document formats.
//
//	`aFormat` The document format(s) available.
func prepFormats(aFormat tPSVstring) *tFormatList {
	list := strings.Split(aFormat, `, `)
	result := make(tFormatList, 0, len(list))
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		result = append(result, TEntity{
			ID:   num,
			Name: a[0],
		})
	}
	if 1 < len(result) {
		sort.Slice(result, func(i, j int) bool {
			return (result[i].Name < result[j].Name) // descending
		})
	}

	return &result
} // prepFormats()

// `prepIdentifiers()` returns a sorted list of document identifiers.
//
//	`aIdentifier` The document's identifier(s).
func prepIdentifiers(aIdentifier tPSVstring) *tIdentifierList {
	list := strings.Split(aIdentifier, `, `)
	result := make(tIdentifierList, 0, len(list))
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least three indices:
		a := append(strings.SplitN(val, `|`, 3), ``)
		num, _ := strconv.Atoi(a[1])
		result = append(result, TEntity{
			ID:   num,
			Name: a[0],
			URL:  a[2],
		})
	}
	if 1 < len(result) {
		sort.Slice(result, func(i, j int) bool {
			return (result[i].Name < result[j].Name) // descending
		})
	}

	return &result
} // prepIdentifiers

// `prepLanguages()` returns a document's languages.
//
//	`aLanguage` The document's language(s).
func prepLanguages(aLanguage tPSVstring) *tLanguageList {
	list := strings.Split(aLanguage, `, `)
	result := make(tLanguageList, 0, len(list))
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		result = append(result, TEntity{
			ID:   num,
			Name: a[0],
		})
	}
	if 1 < len(result) {
		sort.Slice(result, func(i, j int) bool {
			return (result[i].Name < result[j].Name) // descending
		})
	}

	return &result
} // prepLanguages()

var (
	// RegEx to find a document's number of pages.
	quPagesRE = regexp.MustCompile(`(?si)<meta name="calibre:user_metadata:#pages" .*?, &quot;#value#&quot;: (\d+),`)
)

// `prepPages()` prepares the document's `Pages` property.
//
// This functions only returns a value `>0` if the respective `pages`
// plugin is installed with `Calibre` _and_ it uses internally a data
// field called `#pages` stored in the document's metadata file.
//
//	`aPath` The relative directory/path of the document's data.
func prepPages(aPath string) int {
	fName := filepath.Join(dbCalibreLibraryPath, aPath, `metadata.opf`)
	if fi, err := os.Stat(fName); (nil != err) || (0 >= fi.Size()) {
		return 0
	}
	metadata, err := ioutil.ReadFile(fName) // #nosec G304
	if nil != err {
		return 0
	}
	match := quPagesRE.FindSubmatch(metadata)
	if (nil == match) || (1 > len(match)) {
		return 0
	}
	if num, _ := strconv.Atoi(string(match[1])); 0 < num {
		// Since the RegEx returns digits only we don't need
		// to check for errors here.
		return num
	}

	return 0
} // prepPages()

// `prepPublisher()` returns a document's publisher.
//
//	`aPublisher` The document's publisher(s).
func prepPublisher(aPublisher tPSVstring) *tPublisher {
	list := strings.Split(aPublisher, `, `)
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		return &tPublisher{
			ID:   num,
			Name: a[0],
		}
	}

	return nil
} // prepPublisher()

// `prepSeries()` returns a document's series.
//
//	`aSeries` The document's series.
func prepSeries(aSeries tPSVstring) *tSeries {
	list := strings.Split(aSeries, `, `)
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		return &tSeries{
			ID:   num,
			Name: a[0],
		}
	}

	return nil
} // prepSeries()

// `prepTags()` returns a sorted list of document tags.
//
//	`aTag` The document's tag(s).
func prepTags(aTag tPSVstring) *tTagList {
	list := strings.Split(aTag, `, `)
	result := make(tTagList, 0, len(list))
	for _, val := range list {
		if 0 == len(val) {
			continue
		}
		// make sure we have at least two indices:
		a := append(strings.SplitN(val, `|`, 2), ``)
		num, _ := strconv.Atoi(a[1])
		result = append(result, TEntity{
			ID:   num,
			Name: a[0],
		})
	}
	if 1 < len(result) {
		sort.Slice(result, func(i, j int) bool {
			return (result[i].Name < result[j].Name) // descending
		})
	}

	return &result
} // prepTags()

const (
	// see `QueryBy()`, `QuerySearch()`
	quCountQuery = `SELECT COUNT(b.id) FROM books b `
)

// QueryBy returns all documents according to `aOptions`.
//
// The function returns in `rCount` the number of documents found,
// in `rList` either `nil` or a list list of documents,
// in `rErr` either `nil` or the error occurred during the search.
//
//	`aOptions` The options to configure the query.
func QueryBy(aOptions *TQueryOptions) (rCount int, rList *TDocList, rErr error) {
	if rows, err := dbSqliteDB.Query(quCountQuery +
		having(aOptions.Entity, aOptions.ID)); nil == err {
		if rows.Next() {
			_ = rows.Scan(&rCount)
		}
		_ = rows.Close()
	}
	if 0 < rCount {
		rList, rErr = doQueryAll(quBaseQuery +
			having(aOptions.Entity, aOptions.ID) +
			orderBy(aOptions.SortBy, aOptions.Descending) +
			limit(aOptions.LimitStart, aOptions.LimitLength))
	}

	return
} // QueryBy()

const (
	// see `QueryCustomColumns()`
	quCustomColumnsQuery = `SELECT id, label, name, datatype FROM custom_columns `
)

type (
	// TCustomColumn contains info about a user-defined data field.
	TCustomColumn struct {
		ID                    int
		Label, Name, Datatype string
	}

	// TCustomColumnList is a list/slice of `TCustomColumnRec`.
	TCustomColumnList []TCustomColumn
)

// QueryCustomColumns returns data about user-defined columns in `Calibre`.
func QueryCustomColumns() (*TCustomColumnList, error) {
	rows, err := dbSqliteDB.Query(quCustomColumnsQuery)
	if nil != err {
		return nil, err
	}
	defer rows.Close()

	result := make(TCustomColumnList, 0, 8)
	for rows.Next() {
		var cc TCustomColumn
		if err = rows.Scan(&cc.ID, &cc.Label, &cc.Name, &cc.Datatype); nil == err {
			result = append(result, cc)
		}
	}

	return &result, nil
} // QueryCustomColumns()

const (
	// see `QueryDocMini()`
	quDocMiniQuery = `SELECT b.id, IFNULL((SELECT group_concat(d.format, ", ")
FROM data d WHERE d.book = b.id), "") formats,
b.path,
b.title
FROM books b
WHERE b.id = `
)

// QueryDocMini returns the document identified by `aID`.
//
// This function fills only the document properties `ID`, `formats`,
// `path`, and `Title`.
// If a matching document could not be found the function returns `nil`.
//
//	`aID` The document ID to lookup.
func QueryDocMini(aID TID) *TDocument {
	rows, err := dbSqliteDB.Query(quDocMiniQuery +
		strconv.FormatInt(int64(aID), 10))
	if nil != err {
		return nil
	}
	defer rows.Close()

	if rows.Next() {
		var formats tPSVstring
		doc := NewDocument()
		doc.ID = aID
		_ = rows.Scan(&doc.ID, &formats, &doc.path, &doc.Title)
		doc.formats = prepFormats(formats)

		return doc
	}

	return nil
} // QueryDocMini()

// QueryDocument returns the `TDocument` identified by `aID`.
//
// In case the document with `aID` can not be found the function
// returns `nil`.
//
//	`aID` The document ID to lookup.
func QueryDocument(aID TID) *TDocument {
	list, _ := doQueryAll(quBaseQuery +
		`WHERE b.id=` + strconv.FormatInt(int64(aID), 10) +
		` LIMIT 1`)
	if 0 < len(*list) {
		doc := (*list)[0]

		return &doc
	}

	return nil
} // QueryDocument()

const (
	// see `QueryIDs()`
	quIDQuery = `SELECT id, path FROM books `
)

// QueryIDs returns a list of documents with only the `ID` and
// `path` fields set.
//
// This function is used by `thumbnails`.
func QueryIDs() (*TDocList, error) {
	rows, err := dbSqliteDB.Query(quIDQuery)
	if nil != err {
		return nil, err
	}
	defer rows.Close()

	result := NewDocList()
	for rows.Next() {
		doc := NewDocument()
		_ = rows.Scan(&doc.ID, &doc.path)
		result.Add(doc)
	}

	return result, nil
} // QueryIDs()

// QuerySearch returns a list of documents matching the criteria
// in `aOptions`.
//
// The function returns in `rCount` the number of documents found,
// in `rList` either `nil` or a list list of documents,
// in `rErr` either `nil` or an error occurred during the search.
//
//	`aOptions` The options to configure the query.
func QuerySearch(aOptions *TQueryOptions) (rCount int, rList *TDocList, rErr error) {
	where := NewSearch(aOptions.Matching)
	if rows, err := dbSqliteDB.Query(quCountQuery + where.Clause()); nil == err {
		if rows.Next() {
			_ = rows.Scan(&rCount)
		}
		_ = rows.Close()
	}
	if 0 < rCount {
		rList, rErr = doQueryAll(quBaseQuery +
			where.Clause() +
			orderBy(aOptions.SortBy, aOptions.Descending) +
			limit(aOptions.LimitStart, aOptions.LimitLength))
	} else {
		rErr = errors.New(`No documents found`)
	}

	return
} // QuerySearch()

/* _EoF_ */
