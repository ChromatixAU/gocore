package gocore

import (
  "encoding/json"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"

  "github.com/urfave/negroni"
  "github.com/unrolled/render"
)

type Core struct {
  Negroni *negroni.Negroni
  Logger *negroni.Logger
  Static *negroni.Static
  Recovery *negroni.Recovery
  BaseRoute string
  Theme string
  ThemeRenderer *render.Render
  CoreRenderer *render.Render
  Mux *http.ServeMux
  Addr string
  Port string
}

func NewCore() *Core {
  logfilename := os.Getenv( "GO_LOGFILE" )
  if logfilename == "" {
    logfilename = "log/goapp.log"
  }
  errorLog, err := os.OpenFile( logfilename, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666 )
  if err != nil {
    log.Fatal( "error writing to log: " + logfilename )
  }
  defer errorLog.Close()
  theme := os.Getenv( "GO_THEME" )
  if theme == "" {
    log.Fatal("error theme not specified")
  }
  coredir := "github.com/chromatixau/gocore"

  themeRender := render.New( render.Options{ IsDevelopment: true, Directory: theme + "/templates" } )
  coreRender := render.New( render.Options{ IsDevelopment: true, Directory: coredir + "/templates" } )
  mux := http.NewServeMux()
  n := negroni.New()
  l := negroni.NewLoggerWithStream( errorLog )
  r := negroni.NewRecovery()
  r.Logger = l
  r.PrintStack = false
  baseRoute := os.Getenv( "GOBASEROUTE" )

  s := negroni.NewStatic( http.Dir( "public" ) )
  if baseRoute != "" {
    s.Prefix = "/" + baseRoute
  }
  port := ":" + os.Getenv( "PORT" )
  if port == ":" {
    port = ":8080"
  }
  addr := os.Getenv( "SERVER_ADDR" )

  core := Core{
    Negroni: n,
    Logger: l,
    Static: s,
    Recovery: r,
    BaseRoute: baseRoute,
    Theme: theme,
    ThemeRenderer: themeRender,
    CoreRenderer: coreRender,
    Mux: mux,
    Addr: addr,
    Port: port,
  }

  return &core
}

func ( c *Core ) HandleRender() {
  c.Mux.HandleFunc( "/", func( w http.ResponseWriter, req *http.Request ) {
    c.Logger.Println( "start" )
    baseURI, prefix := c.getBaseURI( req )

    templateName, hasTemplate, isPublicFile := c.getTemplate( req, prefix )
    if isPublicFile == true {
      c.Logger.Println( "public file" )
      return
    }
    if false == hasTemplate {
      c.CoreRenderer.HTML( w, http.StatusServiceUnavailable, "templateUnavailable", "" )
      return
    }
    data := c.loadData( req, templateName + ".json", baseURI, prefix )

    c.ThemeRenderer.HTML( w, http.StatusOK, templateName, data )
  } )
}

func ( c *Core ) BindMiddleware() {
  c.Negroni.Use( c.Recovery )
  c.Negroni.Use( c.Logger )
  c.Negroni.UseHandler( c.Mux )
  c.Negroni.Use( c.Static )
}

func ( c *Core ) StartServer() error {
  c.Logger.Println( "Starting Goapp Service" )
  c.Logger.Println( "----------------------" )
  return http.ListenAndServe( c.Addr + c.Port, c.Negroni )
}

func ( c *Core ) getBaseURI( req *http.Request ) ( string, string ) {
  scheme, host, prefix, _ := c.getRequestVars( req )
  baseURI := scheme + "://" + host
  if prefix != "" {
    baseURI = baseURI + "/" + prefix
  }
  return baseURI, prefix
}

func ( c *Core ) getRequestVars( req *http.Request ) ( string, string, string, string ) {
  proto := req.URL.Scheme
  if proto == "" {
    proto = "http"
  }

  forwardedProto := req.Header.Get( "X-Forwarded-Proto" )
  if forwardedProto != "" {
    proto = forwardedProto
  }

  forwardedHost := req.Header.Get( "X-Forwarded-Host" )
  host := req.Host
  if forwardedHost != "" {
    host = forwardedHost
  }

  forwardedPrefix := req.Header.Get( "X-Forwarded-Prefix" )
  prefix := c.BaseRoute
  if forwardedPrefix != "" {
    prefix = forwardedPrefix
  }

  forwardedPath := req.Header.Get( "X-Forwarded-Path" )
  path := req.URL.Path
  if forwardedPath != "" {
    path = forwardedPath
  }
	return proto, host, prefix, path
}

func ( c *Core ) getTemplate( req *http.Request, baseURI string ) ( string, bool, bool ) {
  c.Logger.Println( "RequestURI Template: " + req.RequestURI )
  c.Logger.Println( "BaseURI Template: " + baseURI )
	templateName := strings.TrimSuffix( strings.TrimPrefix( req.RequestURI, "/" + baseURI ), "/" )
  isPublicFile := false
  hasTemplate := false

  if strings.HasPrefix( templateName, "/" ) {
    templateName = strings.TrimPrefix( templateName, "/" )
  }

  c.Logger.Println( "Template Name: [" + templateName + "]" )
	if templateName == "" {
		templateName = "index"
    hasTemplate = c.Exists( templateName, ".tmpl", c.Theme + "/templates" )
    isPublicFile = false
    return templateName, hasTemplate, isPublicFile
	}

  isPublicFile = c.Exists( templateName, "", "public" )
  if false == isPublicFile {
    hasTemplate = c.Exists( templateName, ".tmpl", c.Theme + "/templates" )
  }
  return templateName, hasTemplate, isPublicFile
}

func ( c *Core ) Exists( name string, extension string, folder string ) bool {
  filename := folder + "/" + name + extension
  c.Logger.Println( "file exists?: " + filename )
  _, err := os.Stat( filename )
  if os.IsNotExist( err ) {
    c.Logger.Println( "No" )
    return false
  }
  c.Logger.Println( "Yes" )
  return true
}


func ( c *Core ) loadData( req *http.Request, filename string, baseURI string, prefix string ) interface{} {
  var raw []byte
  var data map[string]interface{}
  c.Logger.Println( "datafile: " + filename )
  raw, err := ioutil.ReadFile( c.Theme + "/data/" + filename )
  if err != nil {
    data = make( map[string]interface{} )
  } else {
    err = json.Unmarshal( raw, &data )
    if err != nil {
      c.Logger.Println( "Unmarshalling JSON failed" )
    }
  }
  data["BaseURI"] = baseURI + "/"
  basePath := ""
  if prefix != "" {
    basePath = "/" + prefix
  }
  path := strings.TrimPrefix( req.RequestURI, basePath )
  data["CanonicalURI"] = strings.TrimSuffix( baseURI + path, "/" )
  return data
}
