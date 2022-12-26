AERC-STYLESETS(7)

# NAME

aerc-stylesets - styleset file specification for *aerc*(1)

# SYNOPSIS

aerc uses a simple configuration syntax to configure the styleset for
its ui.

# STYLESET CONFIGURATION

The styleset is described as *<object>*.*<attribute>* = _<value>_ pairs.

For example, in the line below, the foreground color of the
style object *msglist_unread* is set to _cornflowerblue_

	*msglist_unread*.*fg* = _cornflowerblue_

The configuration also allows wildcard matching of the keys
to configure multiple style objects at a time.

## ATTRIBUTES

The following options are available to be modified for each of the
style objects.

*<object>*.*fg* = _<color>_
	The foreground color of the style object is set.

*<object>*.*bg* = _<color>_
	The background color of the style object is set.

*<object>*.*bold* = _true_|_false_|_toggle_
	The bold attribute of the style object is set/unset.

*<object>*.*blink* = _true_|_false_|_toggle_
	The blink attribute of the style object is set/unset.
	The terminal needs to support blinking text.

*<object>*.*underline* = _true_|_false_|_toggle_
	The underline attribute of the style object is set/unset.
	The terminal needs to support underline text.

*<object>*.*italic* = _true_|_false_|_toggle_
	The italic attribute of the style object is set/unset.
	The terminal needs to support italic text.

*<object>*.*reverse* = _true_|_false_|_toggle_
	Reverses the color of the style object. Exchanges the foreground
	and background colors.

	If the value is _false_, it doesn't change anything.

*<object>*.*normal* = true
	All the attributes of the style object are unset.

	The value doesn't matter.

*<object>*.*default* = true
	Set the style object to the default style of the context. Usually
	based on the terminal.

	The value doesn't matter.

## STYLE OBJECTS

The style objects represent the various ui elements or ui instances for
styling.

[[ *Style Object*
:[ *Description*
|  *default*
:  The default style object used for normal ui elements while not using specialized configuration.
|  *error*
:  The style used to show errors.
|  *warning*
:  The style used when showing warnings.
|  *success*
:  The style used for success messages.
|  *title*
:  The style object used to style titles in ui elements.
|  *header*
:  The style object used to style headers in ui elements.
|  *statusline_default*
:  The default style applied to the statusline.
|  *statusline_error*
:  The style used for error messages in statusline.
|  *statusline_success*
:  The style used for success messages in statusline.
|  *msglist_default*
:  The default style for messages in a message list.
|  *msglist_unread*
:  Unread messages in a message list.
|  *msglist_read*
:  Read messages in a message list.
|  *msglist_flagged*
:  The messages with the flagged flag.
|  *msglist_deleted*
:  The messages marked as deleted.
|  *msglist_marked*
:  The messages with the marked flag.
|  *msglist_result*
:  The messages which match the current search.
|  *dirlist_default*
:  The default style for directories in the directory list.
|  *dirlist_unread*
:  The style used for directories with unread messages
|  *dirlist_recent*
:  The style used for directories with recent messages
|  *completion_default*
:  The default style for the completion engine.
|  *completion_gutter*
:  The completion gutter.
|  *completion_pill*
:  The completion pill.
|  *tab*
:  The style for the tab bar.
|  *stack*
:  The style for ui stack element.
|  *spinner*
:  The style for the loading spinner.
|  *border*
:  The style used to draw borders (only the *bg* color is used unless you customize *border-char-vertical* and/or *border-char-horizontal* in _aerc.conf_).
|  *selector_default*
:  The default style for the selector ui element.
|  *selector_focused*
:  The focused item in a selector ui element.
|  *selector_chooser*
:  The item chooser in a selector ui element.

## FNMATCH STYLE WILDCARD MATCHING

The styleset configuration can be made simpler by using the fnmatch
style wildcard matching for the style object.

The special characters used in the fnmatch wildcards are:

[[ *Pattern*
:[ *Meaning*
|  *\**
:  Matches everything
|  *\?*
:  Matches any single character

For example, the following wildcards can be made using this syntax.

[[ *Example*
:[ *Description*
|  *\**.*fg* = _blue_
:  Set the foreground color of all style objects to blue.
|  *\*list*.*bg* = _hotpink_
:  Set the background color of all style objects that end in list to hotpink.

## SELECTED MODIFIER

The *selected* modifier can be applied to any style object. The style provided for
the *selected* modifier is applied on top of the style object it corresponds to.

If you would like to make sure message that are flagged as read in the msglist
appear in yellow foreground and black background. You can specify that with
this:

	*msglist_default*.*selected*.*fg* = _yellow_

	*msglist_default*.*selected*.*bg* = _black_

If we specify the global style selected modifier using fnmatch as below:

	*\**.*selected*.*reverse* = _toggle_

This toggles the reverse switch for selected version of all the style objects.

## LAYERED STYLES

Some styles, (currently the *msglist_\** and *dirlist_\** ones) are applied in
layers. If a style differs from the base (in this case *\*list_default*) then
that style applies, unless overridden by a higher layer.

The order that *msglist_\** styles are applied in is, from first to last:

. *msglist_default*
. *msglist_unread*
. *msglist_read*
. *msglist_flagged*
. *msglist_deleted*
. *msglist_marked*

So, the marked style will override all other msglist styles.

The order for *dirlist_\** styles is:

. *dirlist_default*
. *dirlist_unread*
. *dirlist_recent*

## COLORS

The color values are set using the values accepted by the tcell library.
The values can be one of the following.

_default_
	The color is set as per the system or terminal default.

_<Color name>_
	Any w3c approved color name is used to set colors for the style.

_<Hex code>_
	Hexcode for a color can be used. The format must be _#XXXXXX_.

_<Dec number>_
	Color based on the terminal palette index. Valid numbers are
	between _0_ and _255_.

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
