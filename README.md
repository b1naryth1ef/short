Shorter - A Go Link Shortner
===========================

> Persistence is a waste of time

This is a link shortner written in Go, my first attempt at writing Go. Things are not very 'Go' like at the moment, and a bit of a mess. 

However, if you need a link shortner that has no persistence, but is very, very fast. you'd be better off writing your own. 

The shortened hash is calculated by base62 encoding the number of links currently stored. As Go deals with UTF-8 strings these have ended up being twice as long as they should. Never mind. 

Also we don't ever store these URLs anywhere, ever, no. Why would you want a to remember about URLs after a chrash or reboot. 

###Use
Set your application domain at the top if the short.go file. It's a variable called domain

####Create
POST request to the root URL with the following JSON <code>{"url":"http://example.com"}</code> the folowing response is sent to you

```JSON
	{"Original":"http://example.com","Short":"AA","FullShort":"http://localhost:8080/AA","HitCount":0}
```


###TODO

[ ] validation of imcoming urls
[ ] increment the hit counter, or ditch it
[ ] show stats for a URL, if the hitcounter stays 
[ ] correct response codes, 201, 404, 500
[ ] better logging of errors
[ ] settings to a settings file