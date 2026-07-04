/* Code Map: Language List
 * - defaultLanguages: 99 ISO 639-1 codes paired with their English
 *   display names. English is first (the IMPL-public-setup §3
 *   requirement). The rest are sorted alphabetically by code.
 * - DefaultLanguageCode: the wizard's default language code ("en").
 *
 * CID Index:
 * CID:wizard-langs-001 -> defaultLanguages
 * CID:wizard-langs-002 -> DefaultLanguageCode
 *
 * Quick lookup: rg -n "CID:wizard-langs-" internal/wizard/steps/
 */
package steps

// CID:wizard-langs-002 - DefaultLanguageCode
// Purpose: the code newState() assigns to Language. Also used as the
// initial active index in the picker. Matches State.Language default.
const DefaultLanguageCode = "en"

// language is one row in the language picker.
type language struct {
	code string // ISO 639-1, e.g. "en", "de"
	name string // English display name, e.g. "English", "German"
}

// CID:wizard-langs-001 - defaultLanguages
// Purpose: the 99 languages the wizard offers. English is the first
// row; the rest are sorted by ISO 639-1 code. The list length and
// ordering are part of the wizard's documented behavior; do not
// re-order without updating the tests and IMPL.
//
// The list is intentionally a Go literal (not loaded from JSON) so the
// wizard has zero file I/O during setup. Adding/removing a row does
// not require any other code change beyond updating this slice.
var defaultLanguages = []language{
	{"en", "English"},
	{"aa", "Afar"},
	{"ab", "Abkhazian"},
	{"af", "Afrikaans"},
	{"ak", "Akan"},
	{"am", "Amharic"},
	{"ar", "Arabic"},
	{"as", "Assamese"},
	{"ay", "Aymara"},
	{"az", "Azerbaijani"},
	{"ba", "Bashkir"},
	{"be", "Belarusian"},
	{"bg", "Bulgarian"},
	{"bh", "Bihari"},
	{"bi", "Bislama"},
	{"bm", "Bambara"},
	{"bn", "Bengali"},
	{"bo", "Tibetan"},
	{"br", "Breton"},
	{"bs", "Bosnian"},
	{"ca", "Catalan"},
	{"ce", "Chechen"},
	{"ch", "Chamorro"},
	{"co", "Corsican"},
	{"cs", "Czech"},
	{"cv", "Chuvash"},
	{"cy", "Welsh"},
	{"da", "Danish"},
	{"de", "German"},
	{"dv", "Dhivehi"},
	{"dz", "Dzongkha"},
	{"ee", "Ewe"},
	{"el", "Greek"},
	{"es", "Spanish"},
	{"et", "Estonian"},
	{"eu", "Basque"},
	{"fa", "Persian"},
	{"ff", "Fulah"},
	{"fi", "Finnish"},
	{"fj", "Fijian"},
	{"fo", "Faroese"},
	{"fr", "French"},
	{"ga", "Irish"},
	{"gd", "Scottish Gaelic"},
	{"gl", "Galician"},
	{"gn", "Guarani"},
	{"gu", "Gujarati"},
	{"ha", "Hausa"},
	{"he", "Hebrew"},
	{"hi", "Hindi"},
	{"hr", "Croatian"},
	{"ht", "Haitian Creole"},
	{"hu", "Hungarian"},
	{"hy", "Armenian"},
	{"ia", "Interlingua"},
	{"id", "Indonesian"},
	{"ig", "Igbo"},
	{"ii", "Sichuan Yi"},
	{"is", "Icelandic"},
	{"it", "Italian"},
	{"iu", "Inuktitut"},
	{"ja", "Japanese"},
	{"jv", "Javanese"},
	{"ka", "Georgian"},
	{"kg", "Kongo"},
	{"ki", "Kikuyu"},
	{"kk", "Kazakh"},
	{"kl", "Kalaallisut"},
	{"km", "Khmer"},
	{"kn", "Kannada"},
	{"ko", "Korean"},
	{"kr", "Kanuri"},
	{"ks", "Kashmiri"},
	{"ku", "Kurdish"},
	{"kv", "Komi"},
	{"kw", "Cornish"},
	{"ky", "Kyrgyz"},
	{"la", "Latin"},
	{"lb", "Luxembourgish"},
	{"lg", "Ganda"},
	{"lo", "Lao"},
	{"lt", "Lithuanian"},
	{"lu", "Luba-Katanga"},
	{"lv", "Latvian"},
	{"mg", "Malagasy"},
	{"mh", "Marshallese"},
	{"mi", "Maori"},
	{"mk", "Macedonian"},
	{"ml", "Malayalam"},
	{"mn", "Mongolian"},
	{"mr", "Marathi"},
	{"ms", "Malay"},
	{"mt", "Maltese"},
	{"my", "Burmese"},
	{"na", "Nauru"},
	{"nb", "Norwegian Bokmal"},
	{"nd", "North Ndebele"},
	{"ne", "Nepali"},
	{"ng", "Ndonga"},
	{"nl", "Dutch"},
	{"nn", "Norwegian Nynorsk"},
	{"no", "Norwegian"},
	{"nr", "South Ndebele"},
	{"ny", "Chichewa"},
	{"oc", "Occitan"},
	{"om", "Oromo"},
}
