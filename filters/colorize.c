/* SPDX-License-Identifier: MIT */
/* Copyright (c) 2023 Robin Jarry */

#include <ctype.h>
#include <fnmatch.h>
#include <getopt.h>
#include <regex.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void usage(void)
{
	puts("usage: colorize [-h] [-8] [-s FILE] [-f FILE]");
	puts("");
	puts("Add terminal escape codes to colorize plain text email bodies.");
	puts("");
	puts("options:");
	puts("  -h       show this help message");
	puts("  -8       emit OSC 8 hyperlink sequences (default $AERC_OSC8_URLS)");
	puts("  -s FILE  use styleset file (default $AERC_STYLESET)");
	puts("  -f FILE  read from filename (default stdin)");
}

enum color_type {
	NONE = 0,
	DEFAULT,
	RGB,
	PALETTE,
};

struct color {
	enum color_type type;
	uint32_t rgb;
	uint32_t index;
};

struct style {
	struct color fg;
	struct color bg;
	bool bold;
	bool blink;
	bool underline;
	bool reverse;
	bool italic;
	bool dim;
	char *sequence;
};

__attribute__((malloc,returns_nonnull))
static void *xmalloc(size_t s)
{
	void *ptr = malloc(s);
	if (ptr == NULL) {
		perror("fatal: cannot allocate buffer");
		abort();
	}
	return ptr;
}

#define BOLD "\x1b[1m"
#define RESET "\x1b[0m"
#define LONGEST_SEQ "\x1b[1;2;3;4;5;7;38;2;255;255;255;48;2;255;255;255m"

static const char *seq(struct style *s) {

	if (!s->sequence) {
		const size_t buf_len = strlen(LONGEST_SEQ) + 1;
		char *buf = xmalloc(buf_len);
		const char *sep = "";
		size_t n = 0;

#define XSPRINTF(...) \
	do { \
		int res = snprintf(buf + n, buf_len - n, __VA_ARGS__); \
		if (res < 0 || (size_t)res >= (buf_len - n)) { \
			perror("fatal: failed to format sequence"); \
			abort(); \
		} \
		n += res; \
	} while (0)

		XSPRINTF("%s", "\x1b[");

		if (s->bold) {
			XSPRINTF("%s1", sep);
			sep = ";";
		}
		if (s->dim) {
			XSPRINTF("%s2", sep);
			sep = ";";
		}
		if (s->italic) {
			XSPRINTF("%s3", sep);
			sep = ";";
		}
		if (s->underline) {
			XSPRINTF("%s4", sep);
			sep = ";";
		}
		if (s->blink) {
			XSPRINTF("%s5", sep);
			sep = ";";
		}
		if (s->reverse) {
			XSPRINTF("%s7", sep);
			sep = ";";
		}
		switch (s->fg.type) {
		case NONE:
			break;
		case DEFAULT:
			XSPRINTF("%s39", sep);
			break;
		case RGB:
			XSPRINTF("%s38;2;%d;%d;%d", sep,
				(s->fg.rgb >> 16) & 0xff,
				(s->fg.rgb >> 8) & 0xff,
				s->fg.rgb & 0xff);
			sep = ";";
			break;
		case PALETTE:
			XSPRINTF(s->fg.index < 8 ?
				"%s3%d" : "%s38;5;%d", sep, s->fg.index);
			sep = ";";
			break;
		}
		switch (s->bg.type) {
		case NONE:
			break;
		case DEFAULT:
			XSPRINTF("%s49", sep);
			break;
		case RGB:
			XSPRINTF("%s48;2;%d;%d;%d", sep,
				(s->bg.rgb >> 16) & 0xff,
				(s->bg.rgb >> 8) & 0xff,
				s->bg.rgb & 0xff);
			break;
		case PALETTE:
			XSPRINTF(s->bg.index < 8 ?
				"%s4%d" : "%s48;5;%d", sep, s->bg.index);
			break;
		}

		if (strcmp(buf, "\x1b[") == 0) {
			XSPRINTF("0");
		}
		XSPRINTF("m");
		s->sequence = buf;
	}
	return s->sequence;
}

struct styles {
	struct style url;
	struct style header;
	struct style signature;
	struct style diff_meta;
	struct style diff_chunk;
	struct style diff_chunk_func;
	struct style diff_add;
	struct style diff_del;
	struct style quote_1;
	struct style quote_2;
	struct style quote_3;
	struct style quote_4;
	struct style quote_x;
};

static FILE *in_file;
static bool osc8_urls;
static const char *styleset;
static struct styles styles = {
	.url = { .underline = true, .fg = { .type = PALETTE, .index = 3 } },
	.header = { .bold = true, .fg = { .type = PALETTE, .index = 4 } },
	.signature = { .dim = true, .fg = { .type = PALETTE, .index = 4 } },
	.diff_meta = { .bold = true },
	.diff_chunk = { .fg = { .type = PALETTE, .index = 6 } },
	.diff_chunk_func = { .dim = true, .fg = { .type = PALETTE, .index = 6 } },
	.diff_add = { .fg = { .type = PALETTE, .index = 2 } },
	.diff_del = { .fg = { .type = PALETTE, .index = 1 } },
	.quote_1 = { .fg = { .type = PALETTE, .index = 6 } },
	.quote_2 = { .fg = { .type = PALETTE, .index = 4 } },
	.quote_3 = { .dim = true, .fg = { .type = PALETTE, .index = 6 } },
	.quote_4 = { .dim = true, .fg = { .type = PALETTE, .index = 4 } },
	.quote_x = { .dim = true, .fg = { .type = PALETTE, .index = 5 } },
};

static inline bool startswith(const char *s, const char *prefix)
{
	return strncmp(s, prefix, strlen(prefix)) == 0;
}

#define ARRAY_SIZE(a) (sizeof(a) / sizeof((a)[0]))

static struct { const char *n; uint32_t c; } color_names[] = {
	{"aliceblue", 0xf0f8ff}, {"antiquewhite", 0xfaebd7}, {"aqua", 0x00ffff},
	{"aquamarine", 0x7fffd4}, {"azure", 0xf0ffff}, {"beige", 0xf5f5dc},
	{"bisque", 0xffe4c4}, {"black", 0x000000}, {"blanchedalmond", 0xffebcd},
	{"blue", 0x0000ff}, {"blueviolet", 0x8a2be2}, {"brown", 0xa52a2a},
	{"burlywood", 0xdeb887}, {"cadetblue", 0x5f9ea0}, {"chartreuse", 0x7fff00},
	{"chocolate", 0xd2691e}, {"coral", 0xff7f50}, {"cornflowerblue", 0x6495ed},
	{"cornsilk", 0xfff8dc}, {"crimson", 0xdc143c}, {"darkblue", 0x00008b},
	{"darkcyan", 0x008b8b}, {"darkgoldenrod", 0xb8860b}, {"darkgray", 0xa9a9a9},
	{"darkgreen", 0x006400}, {"darkkhaki", 0xbdb76b}, {"darkmagenta", 0x8b008b},
	{"darkolivegreen", 0x556b2f}, {"darkorange", 0xff8c00}, {"darkorchid", 0x9932cc},
	{"darkred", 0x8b0000}, {"darksalmon", 0xe9967a}, {"darkseagreen", 0x8fbc8f},
	{"darkslateblue", 0x483d8b}, {"darkslategray", 0x2f4f4f}, {"darkturquoise", 0x00ced1},
	{"darkviolet", 0x9400d3}, {"deeppink", 0xff1493}, {"deepskyblue", 0x00bfff},
	{"dimgray", 0x696969}, {"dodgerblue", 0x1e90ff}, {"firebrick", 0xb22222},
	{"floralwhite", 0xfffaf0}, {"forestgreen", 0x228b22}, {"fuchsia", 0xff00ff},
	{"gainsboro", 0xdcdcdc}, {"ghostwhite", 0xf8f8ff}, {"gold", 0xffd700},
	{"goldenrod", 0xdaa520}, {"gray", 0x808080}, {"green", 0x008000},
	{"greenyellow", 0xadff2f}, {"honeydew", 0xf0fff0}, {"hotpink", 0xff69b4},
	{"indianred", 0xcd5c5c}, {"indigo", 0x4b0082}, {"ivory", 0xfffff0},
	{"khaki", 0xf0e68c}, {"lavender", 0xe6e6fa}, {"lavenderblush", 0xfff0f5},
	{"lawngreen", 0x7cfc00}, {"lemonchiffon", 0xfffacd}, {"lightblue", 0xadd8e6},
	{"lightcoral", 0xf08080}, {"lightcyan", 0xe0ffff}, {"lightgoldenrodyellow", 0xfafad2},
	{"lightgray", 0xd3d3d3}, {"lightgreen", 0x90ee90}, {"lightpink", 0xffb6c1},
	{"lightsalmon", 0xffa07a}, {"lightseagreen", 0x20b2aa}, {"lightskyblue", 0x87cefa},
	{"lightslategray", 0x778899}, {"lightsteelblue", 0xb0c4de}, {"lightyellow", 0xffffe0},
	{"lime", 0x00ff00}, {"limegreen", 0x32cd32}, {"linen", 0xfaf0e6},
	{"maroon", 0x800000}, {"mediumaquamarine", 0x66cdaa}, {"mediumblue", 0x0000cd},
	{"mediumorchid", 0xba55d3}, {"mediumpurple", 0x9370db}, {"mediumseagreen", 0x3cb371},
	{"mediumslateblue", 0x7b68ee}, {"mediumspringgreen", 0x00fa9a}, {"mediumturquoise", 0x48d1cc},
	{"mediumvioletred", 0xc71585}, {"midnightblue", 0x191970}, {"mintcream", 0xf5fffa},
	{"mistyrose", 0xffe4e1}, {"moccasin", 0xffe4b5}, {"navajowhite", 0xffdead},
	{"navy", 0x000080}, {"oldlace", 0xfdf5e6}, {"olive", 0x808000},
	{"olivedrab", 0x6b8e23}, {"orange", 0xffa500}, {"orangered", 0xff4500},
	{"orchid", 0xda70d6}, {"palegoldenrod", 0xeee8aa}, {"palegreen", 0x98fb98},
	{"paleturquoise", 0xafeeee}, {"palevioletred", 0xdb7093}, {"papayawhip", 0xffefd5},
	{"peachpuff", 0xffdab9}, {"peru", 0xcd853f}, {"pink", 0xffc0cb},
	{"plum", 0xdda0dd}, {"powderblue", 0xb0e0e6}, {"purple", 0x800080},
	{"rebeccapurple", 0x663399}, {"red", 0xff0000}, {"rosybrown", 0xbc8f8f},
	{"royalblue", 0x4169e1}, {"saddlebrown", 0x8b4513}, {"salmon", 0xfa8072},
	{"sandybrown", 0xf4a460}, {"seagreen", 0x2e8b57}, {"seashell", 0xfff5ee},
	{"sienna", 0xa0522d}, {"silver", 0xc0c0c0}, {"skyblue", 0x87ceeb},
	{"slateblue", 0x6a5acd}, {"slategray", 0x708090}, {"snow", 0xfffafa},
	{"springgreen", 0x00ff7f}, {"steelblue", 0x4682b4}, {"tan", 0xd2b48c},
	{"teal", 0x008080}, {"thistle", 0xd8bfd8}, {"tomato", 0xff6347},
	{"turquoise", 0x40e0d0}, {"violet", 0xee82ee}, {"wheat", 0xf5deb3},
	{"white", 0xffffff}, {"whitesmoke", 0xf5f5f5}, {"yellow", 0xffff00},
	{"yellowgreen", 0x9acd32},
};

static int color_name(const char *name, uint32_t *color)
{
	for (size_t c = 0; c < ARRAY_SIZE(color_names); c++) {
		if (!strcmp(name, color_names[c].n)) {
			*color = color_names[c].c;
			return 0;
		}
	}
	return 1;
}

static int parse_color(struct color *c, const char *val)
{
	uint32_t color = 0;
	if (!strcmp(val, "default")) {
		c->type = DEFAULT;
	} else if (sscanf(val, "#%x", &color) == 1 && color <= 0xffffff) {
		c->type = RGB;
		c->rgb = color;
	} else if (sscanf(val, "%u", &color) == 1 && color <= 256) {
		c->type = PALETTE;
		c->index = color;
	} else if (!color_name(val, &color)) {
		c->type = RGB;
		c->rgb = color;
	} else {
		fprintf(stderr, "error: invalid color value '%s'\n", val);
		return 1;
	}
	return 0;
}

static int parse_bool(bool *b, const char *val)
{
	if (!strcmp(val, "true")) {
		*b = true;
	} else if (!strcmp(val, "false")) {
		*b = false;
	} else if (!strcmp(val, "toggle")) {
		*b = !*b;
	} else {
		fprintf(stderr, "error: invalid bool value '%s'\n", val);
		return 1;
	}
	return 0;
}

static int set_attr(struct style *s, const char *attr, const char *val)
{
	if (!strcmp(attr, "fg")) {
		if (parse_color(&s->fg, val))
			return 1;
	} else if (!strcmp(attr, "bg")) {
		if (parse_color(&s->fg, val))
			return 1;
	} else if (!strcmp(attr, "bold")) {
		if (parse_bool(&s->bold, val))
			return 1;
	} else if (!strcmp(attr, "blink")) {
		if (parse_bool(&s->blink, val))
			return 1;
	} else if (!strcmp(attr, "underline")) {
		if (parse_bool(&s->underline, val))
			return 1;
	} else if (!strcmp(attr, "reverse")) {
		if (parse_bool(&s->reverse, val))
			return 1;
	} else if (!strcmp(attr, "italic")) {
		if (parse_bool(&s->italic, val))
			return 1;
	} else if (!strcmp(attr, "dim")) {
		if (parse_bool(&s->dim, val))
			return 1;
	} else if (!strcmp(attr, "normal")) {
		s->bold = false;
		s->underline = false;
		s->reverse = false;
		s->italic = false;
		s->dim = false;
	} else if (!strcmp(attr, "default")) {
		s->fg.type = NONE;
		s->fg.type = NONE;
	} else {
		fprintf(stderr, "error: invalid style attribute '%s'\n", attr);
		return 1;
	}
	return 0;
}

static struct {const char *n; struct style *s;} ini_objects[] = {
	{"url", &styles.url},
	{"header", &styles.header},
	{"signature", &styles.signature},
	{"diff_meta", &styles.diff_meta},
	{"diff_chunk", &styles.diff_chunk},
	{"diff_chunk_func", &styles.diff_chunk_func},
	{"diff_add", &styles.diff_add},
	{"diff_del", &styles.diff_del},
	{"quote_1", &styles.quote_1},
	{"quote_2", &styles.quote_2},
	{"quote_3", &styles.quote_3},
	{"quote_4", &styles.quote_4},
	{"quote_x", &styles.quote_x},
};

/*                         object            attribute           value */
#define STYLE_LINE_FORMAT "%127[0-9A-Za-z_-*?].%127[0-9a-zA-Z_-] = %127[#a-zA-Z0-9]s"

static int parse_styleset(void)
{
	bool in_section = false;
	char buf[BUFSIZ];
	int err = 0;
	FILE *f;

	if (!styleset)
		return 0;

	f = fopen(styleset, "r");
	if (!f) {
		perror("error: failed to open styleset");
		return 1;
	}

	while (fgets(buf, sizeof(buf), f)) {
		/* strip LF, CR, CRLF, LFCR */
		buf[strcspn(buf, "\r\n")] = '\0';
		if (in_section) {
			char obj[128], attr[128], val[128];
			bool changed = false;

			if (sscanf(buf, STYLE_LINE_FORMAT, obj, attr, val) != 3) {
				if (buf[0] == '[') {
					/* start of another section */
					break;
				}
				continue;
			}

			for (size_t o = 0; o < ARRAY_SIZE(ini_objects); o++) {
				if (fnmatch(obj, ini_objects[o].n, 0))
					continue;
				if (set_attr(ini_objects[o].s, attr, val)) {
					err = 1;
					goto end;
				}
				changed = true;
			}
			if (!changed) {
				fprintf(stderr,
					"error: unknown style object %s\n",
					obj);
				err = 1;
				goto end;
			}
		} else if (!strcmp(buf, "[viewer]")) {
			in_section = true;
		}
	}

end:
	fclose(f);
	return err;
}

static inline void print(const char *in)
{
	fputs(in, stdout);
}

static inline size_t print_notabs(const char *in, size_t max_len)
{
	size_t len = 0;
	while (*in != '\0' && len < max_len) {
		char c = *in++;
		if (c == '\t') {
			/* Tabs are interpreted as cursor movement and are not
			 * colored like regular characters. Replace them with
			 * 8 spaces. */
			fputs("        ", stdout);
		} else {
			fputc(c, stdout);
		}
		len++;
	}
	return len;
}

static void print_osc8(const char *url, size_t len, size_t id, bool email) {
	print("\x1b]8;");
	if (url != NULL) {
		printf("id=colorize-%lu;", id);
		if (email) {
			print("mailto://");
		}
		print_notabs(url, len);
	} else {
		/* do not print and url id for the terminator */
		print(";");
	}
	print("\x1b\\");
}

static void diff_chunk(const char *in)
{
	size_t len = 0;
	print(seq(&styles.diff_chunk));
	while (in[len] == '@')
		len++;
	while (in[len] != '\0' && in[len] != '@')
		len++;
	while (in[len] == '@')
		len++;
	in += print_notabs(in, len);
	print(RESET);
	print(seq(&styles.diff_chunk_func));
	print_notabs(in, BUFSIZ);
	print(RESET);
}

static inline bool isurichar(char c)
{
	if (c == '\0')
		return false;
	if (isalnum(c))
		return true;
	if (strchr("-_.,~:;/?#@!$&%*+=\"'|<>()[]", c) != NULL)
		return true;
	return false;
}

#define URL_RE \
	"([a-z]{2,8})://" \
	"|(mailto:)?[[:alnum:]_+.~/-]*[[:alnum:]]@[[:alnum:]][[:alnum:].-]*[[:alnum:]]"
static regex_t url_re;

static void urls(const char *in, struct style *ctx)
{
	/* ID of the next link to print for OSC 8. The purpose of passing
	 * explicit ID is to help terminal emulator with grouping of
	 * multi-line links in nested terminal sessions */
	static size_t url_id = 0;

	regmatch_t groups[3];
	size_t len;
	bool trim;

	while (!regexec(&url_re, in, 3, groups, 0)) {
		in += print_notabs(in, (size_t)groups[0].rm_so);
		len = (size_t)groups[0].rm_eo - (size_t)groups[0].rm_so;

		if (groups[1].rm_so != -1) {
			/* Standard URL (i.e. not mailto: nor email address).
			 * Regular expressions do not really cut it here and
			 * we need to detect opening/closing braces to handle
			 * markdown link syntax. */
			int paren = 0, bracket = 0, ltgt = 0;
			bool emit_url = false;
			size_t l = len;

			while (!emit_url && isurichar(in[l])) {
				switch (in[l]) {
				case '[': bracket++; l++; break;
				case '(': paren++; l++; break;
				case '<': ltgt++; l++; break;
				case ']':
					if (--bracket < 0)
						emit_url = true;
					else
						l++;
					break;
				case ')':
					if (--paren < 0)
						emit_url = true;
					else
						l++;
					break;
				case '>':
					if (--ltgt < 0)
						emit_url = true;
					else
						l++;
					break;
				default:
					l++;
					break;
				}
			}
			/* Heuristic to remove trailing characters that are
			 * valid URL characters, but typically not at the end
			 * of the URL */
			trim = true;
			while (trim && l > len) {
				switch (in[l - 1]) {
				case '.': case ',': case ':':
				case ';': case '?': case '!':
				case '"': case '\'': case '%':
					l--;
					break;
				default:
					trim = false;
					break;
				}
			}
			if (l == len) {
				/* only an URL protocol, do not colorize */
				in += print_notabs(in, len);
				continue;
			}
			len = l;
		}
		print(seq(&styles.url));
		bool email = groups[2].rm_so == -1 && groups[1].rm_so == -1;
		if (osc8_urls) {
			print_osc8(in, len, url_id, email);
		}
		in += print_notabs(in, len);
		if (osc8_urls) {
			print_osc8(NULL, 0, url_id, email);
		}
		url_id++;
		print(RESET);
		if (ctx) {
			print(seq(ctx));
		}
	}
	print_notabs(in, BUFSIZ);
}

static inline void signature(const char *in)
{
	print(seq(&styles.signature));
	urls(in, &styles.signature);
	print(RESET);
}

#define HEADER_RE "^[A-Z][[:alnum:]_-]+:"
static regex_t header_re;

static void header(const char *in)
{
	regmatch_t groups[1];

	if (!regexec(&header_re, in, 1, groups, 0)) {
		print(seq(&styles.header));
		in += print_notabs(in, (size_t)groups[0].rm_eo);
		print(RESET);
	}
	urls(in, NULL);
}

#define DIFF_START_RE "^(diff (--git|-up|-u)|---) [[:graph:]]"
static regex_t diff_start_re;

#define DIFF_META_RE \
	"^(diff (--git|-up|-u)|(new|deleted) file|similarity" \
	" index|(rename|copy) (to|from)|index|---|\\+\\+\\+) "
static regex_t diff_meta_re;

static void quote(const char *in)
{
	regmatch_t groups[8];
	struct style *s;
	size_t q, level;

	q = level = 0;
	while (in[q] == '>') {
		level++;
		q++;
		if (in[q] == ' ')
			q++;
	}
	switch (level) {
	case 1:
		s = &styles.quote_1;
		break;
	case 2:
		s = &styles.quote_2;
		break;
	case 3:
		s = &styles.quote_3;
		break;
	case 4:
		s = &styles.quote_4;
		break;
	default:
		s = &styles.quote_x;
		break;
	}

	print(seq(s));
	in += print_notabs(in, q);
	if (startswith(in, "+")) {
		printf("%s%s", RESET, seq(&styles.diff_add));
		print_notabs(in, BUFSIZ);
	} else if (startswith(in, "-")) {
		printf("%s%s", RESET, seq(&styles.diff_del));
		print_notabs(in, BUFSIZ);
	} else if (!regexec(&diff_meta_re, in, 8, groups, 0)) {
		print(BOLD);
		print_notabs(in, BUFSIZ);
	} else {
		urls(in, s);
	}
	print(RESET);
}

static void print_style(const char *in, struct style *s)
{
	print(seq(s));
	print_notabs(in, BUFSIZ);
	print(RESET);
}

enum state { INIT, DIFF, SIGNATURE, BODY };

static void colorize_line(const char *in)
{
	static enum state state = INIT;
	regmatch_t groups[8];  /* enough groups to cover all expressions */

	switch (state) {
	case DIFF:
		if (!strcmp(in, "-- ")) {
			state = SIGNATURE;
			signature(in);
		} else if (startswith(in, "@@ ")) {
			diff_chunk(in);
		} else if (!regexec(&diff_meta_re, in, 8, groups, 0)) {
			print_style(in, &styles.diff_meta);
		} else if (startswith(in, "+")) {
			print_style(in, &styles.diff_add);
		} else if (startswith(in, "-")) {
			print_style(in, &styles.diff_del);
		} else if (!startswith(in, " ") && strcmp(in, "") != 0) {
			state = BODY;
			if (startswith(in, ">")) {
				quote(in);
			} else {
				urls(in, NULL);
			}
		} else {
			print_notabs(in, BUFSIZ);
		}
		break;
	case SIGNATURE:
		signature(in);
		break;
	default: /* BODY, INIT */
		if (!regexec(&diff_start_re, in, 8, groups, 0)) {
			state = DIFF;
			print_style(in, &styles.diff_meta);
		} else if (!strcmp(in, "-- ")) {
			state = SIGNATURE;
			signature(in);
		} else {
			state = BODY;
			if (startswith(in, ">")) {
				quote(in);
			} else if (!regexec(&header_re, in, 8, groups, 0)) {
				header(in);
			} else {
				urls(in, NULL);
			}
		}
		break;
	}
}

static int parse_args(int argc, char **argv)
{
	const char *filename = NULL, *osc8 = NULL;
	int c;

	styleset = getenv("AERC_STYLESET");
	osc8 = getenv("AERC_OSC8_URLS");

	while ((c = getopt(argc, argv, "h8s:f:")) != -1) {
		switch (c) {
		case '8':
			osc8 = "1";
			break;
		case 's':
			styleset = optarg;
			break;
		case 'f':
			filename = optarg;
			break;
		default:
			usage();
			return 1;
		}
	}
	if (optind < argc) {
		fprintf(stderr, "%s: unexpected argument -- '%s'\n",
			argv[0], argv[optind]);
		usage();
		return 1;
	}
	if (filename == NULL || !strcmp(filename, "-")) {
		in_file = stdin;
	} else {
		in_file = fopen(filename, "r");
		if (!in_file) {
			perror("error: cannot open file");
			return 1;
		}
	}
	osc8_urls = osc8 != NULL;

	return 0;
}

int main(int argc, char **argv)
{
	char buf[BUFSIZ];
	int err;

	regcomp(&header_re, HEADER_RE, REG_EXTENDED);
	regcomp(&diff_start_re, DIFF_START_RE, REG_EXTENDED);
	regcomp(&diff_meta_re, DIFF_META_RE, REG_EXTENDED);
	regcomp(&url_re, URL_RE, REG_EXTENDED);

	err = parse_args(argc, argv);
	if (err) {
		goto end;
	}
	err = parse_styleset();
	if (err) {
		goto end;
	}
	while (fgets(buf, sizeof(buf), in_file)) {
		/* strip LF, CR, CRLF, LFCR */
		buf[strcspn(buf, "\r\n")] = '\0';
		colorize_line(buf);
		printf("\n");
	}
end:
	if (in_file) {
		fclose(in_file);
	}
	return err;
}
