aerc-stylesets(7)

# Name

aerc-stylesets - styleset file specification for *aerc*(1)

# SYNOPSIS

aerc uses a simple configuration syntax to configure the styleset for
its ui.

# Styleset Configuration

Aerc uses a simple configuration file to describe a styleset. The
styleset is described as key, value pairs. In each line, the key
represents the style object it signifies and the color/atrribute of
that is modified.

For example, in the line below, the foreground color of the
style object "msglist_unread" is set to "cornflowerblue"
```
msglist_unread.fg=cornflowerblue
```

The configuration also allows wildcard matching of the style_objects
to configure multiple style objects at a time.

## Style
The following options are available to be modified for each of the
style objects.

*fg*
	The foreground color of the style object is set.

	Syntax: `<style_object>.fg=<color>`

*bg*
	The background color of the style object is set.

	Syntax: `<style_object>.bg=<color>`

*bold*
	The bold attribute of the style object is set/unset.

	Syntax: `<style_object>.bold=<true|false|toggle>`

*blink*
	The blink attribute of the style object is set/unset.
	_The terminal needs to support blinking text_

	Syntax: `<style_object>.bold=<true|false|toggle>`

*underline*
	The underline attribute of the style object is set/unset.
	_The terminal needs to support underline text_

	Syntax: `<style_object>.underline=<true|false|toggle>`

*reverse*
	Reverses the color of the style object. Exchanges the foreground
	and background colors.

	Syntax: `<style_object>.reverse=<true|false|toggle>`
	_If the value is false, it doesn't change anything_

*normal*
	All the attributes of the style object are unset.

	Syntax: `<style_object>.normal=<true>`
	_The value doesn't matter_

*default*
	Set the style object to the default style of the context. Usually
	based on the terminal.

	Syntax: `<style_object>.default=<true>`
	_The value doesn't matter_

## Style Objects
The style objects represent the various ui elements or ui instances for
styling.

[[ *Style Object*
:[ *Description*
|  default
:  The default style object used for normal ui elements while not
   using specialized configuration.
|  error
:  The style used to show errors.
|  warning 
:  The style used when showing warnings.
|  success
:  The style used for success messages.
|  title
:  The style object used to style titles in ui elements.
|  header
:  The style object used to style headers in ui elements.
|  statusline_default
:  The default style applied to the statusline.
|  statusline_error
:  The style used for error messages in statusline.
|  statusline_success
:  The style used for success messages in statusline.
|  msglist_default
:  The default style for messages in a message list.
|  msglist_unread
:  Unread messages in a message list.
|  msglist_read
:  Read messages in a message list.
|  msglist_deleted
:  The messages marked as deleted.
|  msglist_marked
:  The messages with the marked flag.
|  msglist_flagged
:  The messages with the flagged flag.
|  dirlist_default
:  The default style for directories in the directory list.
|  completion_default
:  The default style for the completion engine.
|  completion_gutter
:  The completion gutter.
|  completion_pill
:  The completion pill.
|  tab
:  The style for the tab bar.
|  stack
:  The style for ui stack element.
|  spinner
:  The style for the loading spinner.
|  border
:  The style used to draw borders. *Only the background color is used*.
|  selector_default
:  The default style for the selector ui element.
|  selector_focused
:  The focused item in a selector ui element.
|  selector_chooser
:  The item chooser in a selector ui element.

## fnmatch style wildcard matching
The styleset configuration can be made simpler by using the fnmatch
style wildcard matching for the style object.

The special characters used in the fnmatch wildcards are:
[[ *Pattern*
:[ *Meaning*
|  \*
:  Matches everything
|  \?
:  Matches any single character

For example, the following wildcards can be made using this syntax.
[[ *Example*
:[ Description
|  \*.fg=blue
:  Set the foreground color of all style objects to blue.
|  \*list.bg=hotpink
:  Set the background color of all style objects that end in list 
   to hotpink.

## Selected modifier
Selected modifier can be applied to any style object. The style provided for
the selected modifier are applied on top of the style object it corresponds to.

If you would like to make sure message that are flagged as read in the msglist
appear in yellow foreground and black background. You can specify that with
this:

	msglist_default.selected.fg=yellow

	msglist_default.selected.bg=black

If we specify the global style selected modifer using fnmatch as below:

	\*.selected.reverse=toggle

This toggles the reverse switch for selected version of all the style objects.

## Colors
The color values are set using the values accepted by the tcell library.
The values can be one of the following.

	*default*
		The color is set as per the system or terminal default.

	*<Color name>*
		Any w3c approved color name is used to set colors for the style.

	*<Hex code>*
		Hexcode for a color can be used. The format must be "\#XXXXXX"

