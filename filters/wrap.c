/* SPDX-License-Identifier: MIT */
/* Copyright (c) 2023 Robin Jarry */

#define _XOPEN_SOURCE
#include <errno.h>
#include <getopt.h>
#include <locale.h>
#include <regex.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>
#include <wctype.h>

static void usage(void)
{
	puts("usage: wrap [-h] [-w INT] [-r] [-l INT] [-f FILE]");
	puts("");
	puts("Wrap text without messing up email quotes.");
	puts("");
	puts("options:");
	puts("  -h       show this help message");
	puts("  -w INT   preferred wrap margin (default 80)");
	puts("  -r       reflow all paragraphs even if no trailing space");
	puts("  -l INT   minimum percentage of letters in a line to be");
	puts("           considered a paragaph");
	puts("  -f FILE  read from filename (default stdin)");
}

static size_t margin = 80;
static long prose_ratio = 50;
static bool reflow;
static FILE *in_file;

int parse_args(int argc, char **argv)
{
	const char *filename = NULL;
	char c;

	while ((c = getopt(argc, argv, "hrw:l:f:")) != -1) {
		errno = 0;
		switch (c) {
		case 'r':
			reflow = true;
			break;
		case 'l':
			prose_ratio = strtol(optarg, NULL, 10);
			if (errno) {
				perror("error: invalid ratio value");
				return 1;
			}
			if (prose_ratio <= 0 || prose_ratio >= 100) {
				fprintf(stderr, "error: ratio must be ]0,100[\n");
				return 1;
			}
			break;
		case 'w':
			margin = strtol(optarg, NULL, 10);
			if (errno) {
				perror("error: invalid width value");
				return 1;
			}
			if (margin < 1) {
				fprintf(stderr, "error: width must be positive\n");
				return 1;
			}
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
	return 0;
}

static bool is_empty(const wchar_t *s)
{
	while (*s != L'\0') {
		if (!iswspace(*s++))
			return false;
	}
	return true;
}

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

__attribute__((malloc,returns_nonnull))
static void *xrealloc(void *ptr, size_t s)
{
	ptr = realloc(ptr, s);
	if (ptr == NULL) {
		perror("fatal: cannot reallocate buffer");
		abort();
	}
	return ptr;
}

struct paragraph {
	/* email quote prefix, if any */
	wchar_t *quotes;
	/* list item indent, if any */
	wchar_t *indent;
	/* actual text of this paragraph */
	wchar_t *text;
	/* percentage of letters in text */
	int prose_ratio;
	/* text ends with a space */
	bool flowed;
	/* paragraph is a list item */
	bool list_item;
};

static void free_paragraph(struct paragraph *p)
{
	if (!p)
		return;
	free(p->quotes);
	free(p->indent);
	free(p->text);
	free(p);
}

static wchar_t *read_part(const wchar_t *in, size_t len)
{
	wchar_t *out = xmalloc((len + 1) * sizeof(wchar_t));
	wcsncpy(out, in, len);
	out[len] = L'\0';
	return out;
}

static size_t list_item_offset(const wchar_t *buf)
{
	size_t i = 0;
	wchar_t c;

	if (buf[i] == L'-' || buf[i] == '*' || buf[i] == '.') {
		/* bullet list */
		i++;
	} else if (iswdigit(buf[i])) {
		/* numbered list */
		i++;
		if (iswdigit(buf[i])) {
			i++;
		}
	} else if (iswalpha(buf[i])) {
		/* lettered list */
		c = towlower(buf[i]);
		i++;
		if (c == L'i' || c == L'v') {
			/* roman i. ii. iii. iv. ... */
			c = towlower(buf[i]);
			while (i < 4 && (c == L'i' || c == L'v')) {
				c = towlower(buf[++i]);
			}
		}
	} else {
		return 0;
	}
	if (iswdigit(buf[0]) || iswalpha(buf[0])) {
		if (buf[i] == L')' || buf[i] == L'/' || buf[i] == L'.') {
			i++;
		} else {
			return 0;
		}
	}
	if (buf[i] == L' ') {
		i++;
	} else {
		return 0;
	}

	return i;
}

static struct paragraph *parse_line(const wchar_t *buf)
{
	size_t i, q, t, e, letters, indent_len, text_len;
	bool list_item, flowed;
	struct paragraph *p;

	/*
	 * Find relevant positions in the line:
	 *
	 * '> > > >       2)       blah blah blah blah    '
	 *  ^       ^              ^                  ^
	 *  0       q              t                  e
	 *  <------><------------->
	 *   quotes     indent
	 *          <-------------------------------->
	 *                        text
	 */

	/* detect the end of quotes prefix if any */
	q = 0;
	while (buf[q] == L'>') {
		q++;
		if (buf[q] == L' ') {
			q++;
		}
	}
	/* detect list item prefix & indent */
	t = q;
	while (iswspace(buf[t])) {
		t++;
	}
	i = list_item_offset(&buf[t]);
	list_item = i != 0;
	t += i;
	while (iswspace(buf[t])) {
		t++;
	}
	indent_len = t - q;
	/* compute prose ratio */
	e = t;
	letters = 0;
	while (buf[e] != L'\0') {
		if (iswalpha(buf[e++])) {
			letters++;
		}
	}
	/* strip trailing whitespace */
	flowed = false;
	while (e > q && iswspace(buf[e - 1])) {
		e--;
		flowed = true;
	}
	text_len = e - q;

	p = xmalloc(sizeof(*p));
	memset(p, 0, sizeof(*p));
	p->quotes = read_part(buf, q);
	p->indent = xmalloc((indent_len + 1) * sizeof(wchar_t));
	for (i = 0; i < indent_len; i++)
		p->indent[i] = L' ';
	p->indent[i] = L'\0';
	p->text = read_part(&buf[q], text_len);
	p->flowed = flowed;
	p->list_item = list_item;
	p->prose_ratio = 100 * letters / (text_len ? text_len : 1);

	return p;
}

static bool is_continuation(
	const struct paragraph *p, const struct paragraph *next
) {
	if (next->list_item)
		/* new list items always start a new paragraph */
		return false;
	if (next->prose_ratio < prose_ratio || p->prose_ratio < prose_ratio)
		/* does not look like prose, maybe ascii art */
		return false;
	if (wcscmp(next->quotes, p->quotes) != 0)
		/* quote prefix has changed */
		return false;
	if (wcscmp(next->indent, p->indent) != 0)
		/* list item indent has changed */
		return false;
	if (is_empty(next->text))
		/* empty or whitespace only line */
		return false;
	if (wcscmp(p->text, L"--") == 0)
		/* never join anything with signature start */
		return false;
	if (p->flowed)
		/* current paragraph has trailing space, indicating
		 * format=flowed */
		return true;
	if (reflow)
		/* user forced paragraph reflow on the command line */
		return true;
	return false;
}

static void join_paragraph(
	struct paragraph *p, const struct paragraph *next
) {
	const wchar_t *append = next->text;
	const wchar_t *separator = L" ";
	size_t len, extra_len;
	wchar_t *text;

	/* trim leading whitespace of the next paragraph before joining */
	while (*append != L'\0' && iswspace(*append))
		append++;

	len = wcslen(p->text);
	if (len == 0) {
		separator = L"";
	}
	extra_len = wcslen(separator) + wcslen(append) + 1;

	text = xrealloc(p->text, (len + extra_len) * sizeof(wchar_t));
	swprintf(&text[len], extra_len, L"%ls%ls", separator, append);

	p->text = text;
	p->prose_ratio = (p->prose_ratio + next->prose_ratio) / 2;
	p->flowed = next->flowed;
}

/*
 * BUFSIZ has different values depending on the libc implementation.
 * Use a self defined value to have consistent behaviour accross all platforms.
 */
#define BUFFER_SIZE 8192

/*
 * Write a paragraph, wrapping at words boundaries.
 *
 * Only try to do word wrapping on things that look like prose. When the text
 * contains too many non-letter characters, print it as-is.
 */
static void write_paragraph(struct paragraph *p)
{
	size_t quotes_width = wcswidth(p->quotes, wcslen(p->quotes));
	size_t remain = wcswidth(p->text, wcslen(p->text));
	const wchar_t *indent = L"";
	wchar_t *text = p->text;
	bool more = true;
	wchar_t *line;
	size_t width;

	while (more) {
		width = quotes_width + wcswidth(indent, wcslen(indent));

		if (width + remain <= margin || p->prose_ratio < prose_ratio) {
			/* whole paragraph fits on a single line */
			line = text;
			more = false;
		} else {
			/* find split point, preferably before margin */
			int split = -1;
			int w = 0;
			for (int i = 0; text[i] != L'\0'; i++) {
				w += wcwidth(text[i]);
				if (width + w > margin && split != -1) {
					break;
				}
				if (iswspace(text[i])) {
					split = i;
				}
			}
			if (split == -1) {
				/* no space found to split, print a long line */
				line = text;
				more = false;
			} else {
				text[split] = L'\0';
				line = text;
				split++;
				/* find start of next word */
				while (iswspace(text[split])) {
					split++;
				}
				if (text[split] != L'\0') {
					text = &text[split];
					remain -= split;
				} else {
					/* only trailing whitespace, we're done */
					more = false;
				}
			}
		}
		wprintf(L"%ls%ls%ls\n", p->quotes, indent, line);
		indent = p->indent;
	}
}

#define SPACES_PER_TAB 8

/*
 * Trim LF CR CRLF LFCR and replace tabs with spaces.
 */
static void sanitize_line(const wchar_t *in, wchar_t *out)
{
	/* No bounds checking needed. This function is only used with
	 * 'buf' and 'line' buffers from main. 'out' is large enough no
	 * matter what is present in 'in'. */
	while (*in != L'\0' && *in != L'\n' && *in != L'\r') {
		if (*in == L'\t') {
			/* tabs cause indentation/alignment issues
			 * replace them with 8 spaces */
			in++;
			for (int i = 0; i < SPACES_PER_TAB; i++)
				*out++ = L' ';
		} else {
			*out++ = *in++;
		}
	}
	*out = L'\0';
}

int main(int argc, char **argv)
{
	/* line needs to be 8 times larger than buf since every read character
	 * may be a tab (very unlikely, but it could happen). */
	static wchar_t buf[BUFFER_SIZE], line[BUFFER_SIZE * SPACES_PER_TAB];
	struct paragraph *cur = NULL, *next;
	bool is_patch = false;
	regmatch_t groups[2];
	char *subject;
	regex_t re;
	int err;

	err = parse_args(argc, argv);
	if (err)
		goto end;

	regcomp(&re, "\\<PATCH\\>", REG_EXTENDED);
	subject = getenv("AERC_SUBJECT");
	if (subject && !regexec(&re, subject, 2, groups, 0))
		is_patch = true;
	regfree(&re);

	/* aerc will always send UTF-8 text, force locale here */
	if (!setlocale(LC_CTYPE, "C.UTF-8")) {
		err = 1;
		perror("error: failed to set locale");
		goto end;
	}
	fwide(in_file, true);
	fwide(stdout, true);

	while (fgetws(buf, BUFFER_SIZE, in_file)) {
		if (is_patch) {
			/* never reflow patches */
			fputws(buf, stdout);
			continue;
		}
		sanitize_line(buf, line);
		next = parse_line(line);
		if (!cur) {
			cur = next;
		} else if (is_continuation(cur, next)) {
			join_paragraph(cur, next);
			free_paragraph(next);
		} else {
			write_paragraph(cur);
			free_paragraph(cur);
			cur = next;
		}
	}
	if (cur) {
		write_paragraph(cur);
	}

end:
	free_paragraph(cur);
	if (in_file) {
		fclose(in_file);
	}
	return err;
}
