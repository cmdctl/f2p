package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	uploadDir = os.TempDir()
	fileMap   sync.Map
)

func main() {
	http.HandleFunc("/", uploadPage)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download/", downloadPage)
	http.HandleFunc("/file/", downloadHandler)

	log.Println("💾 f2p running at http://localhost:9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}

// ===============================
// Upload Page (Full-screen terminal look)
// ===============================
func uploadPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="theme-color" content="#00ff66">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<title>f2p • file-to-peer</title>
<style>
html,body {
  height:100%; margin:0;
  background:#000;
  color:#00ff66;
  font-family:"Courier New", monospace;
  display:flex; flex-direction:column;
  justify-content:center; align-items:center;
  text-align:center;
  /* For iPhone safe-area insets */
  padding-top:env(safe-area-inset-top);
  padding-bottom:env(safe-area-inset-bottom);
  background-attachment:fixed;
}
h1 {
  font-weight:normal;
  font-size:2rem;
  color:#00ff66;
  margin-bottom:1rem;
}
@media (max-width:500px){
  h1 { font-size:1.5rem; }
}
.blink::after {
  content:'▮';
  animation:blink 1s steps(1,end) infinite;
}
@keyframes blink { 50%{opacity:0;} }
input[type=file] {
  color:#00ff66;
  background:transparent;
  border:none;
  font-family:inherit;
  font-size:1rem;
  margin:1rem 0;
  width:80%;
  text-align:center;
}
button {
  background:#00ff66;
  color:#000;
  border:none;
  padding:0.6rem 1.4rem;
  cursor:pointer;
  font-family:inherit;
  border-radius:4px;
  font-size:1rem;
  width:60%;
  max-width:300px;
}
button:hover { background:#00cc55; }
progress {
  width:80%;
  height:8px;
  background:#111;
  border:none;
  margin:1rem 0;
}
progress::-webkit-progress-value { background:#00ff66; }
a { color:#00ff66; word-break:break-all; text-decoration:none; }
.copybtn {
  background:transparent;
  border:1px solid #00ff66;
  color:#00ff66;
  padding:0.4rem 0.8rem;
  border-radius:4px;
  cursor:pointer;
  margin-top:0.5rem;
}
.copybtn:hover { background:#00ff66; color:#000; }
footer {
  position:fixed;
  bottom:1rem;
  font-size:0.8rem;
  color:#008844;
}
</style>
</head>
<body>
<h1>f2p<span class="blink"></span></h1>
<p>Upload a file to generate a one-time peer link.</p>
<input type="file" id="file"><br>
<button onclick="upload()">Upload</button>
<progress id="prog" value="0" max="100"></progress>
<pre id="msg"></pre>
<footer>file-to-peer © 2025</footer>
<script>
function upload(){
  const f=document.getElementById('file').files[0];
  if(!f){alert('Select a file first');return;}
  const xhr=new XMLHttpRequest();
  xhr.open('POST','/upload',true);
  xhr.upload.onprogress=(e)=>{
    if(e.lengthComputable)
      document.getElementById('prog').value=(e.loaded/e.total)*100;
  };
  xhr.onload=()=>{
    const msg=document.getElementById('msg');
    if(xhr.status===200){
      const url=xhr.responseText.trim();
      msg.innerHTML="<br/>Upload complete!<br/><br/>Link: "+url+
        "<br/><br/><button class='copybtn' onclick='copyLink(\""+url+"\")'>Copy Link</button>";
    } else msg.textContent="Error: "+xhr.responseText;
  };
  const fd=new FormData(); fd.append('file',f); xhr.send(fd);
}
function copyLink(link){
  navigator.clipboard.writeText(link);
  alert("Copied to clipboard:\n"+link);
}
</script>
</body></html>`)
}

// ===============================
// Upload Handler
// ===============================
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if part.FileName() == "" {
			continue
		}

		id := randomID(8)
		original := filepath.Base(part.FileName())
		dstPath := filepath.Join(uploadDir, id+"_"+original)

		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Cannot save file", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(dst, part); err != nil {
			dst.Close()
			http.Error(w, "Write error", http.StatusInternalServerError)
			return
		}
		dst.Close()

		fileMap.Store(id, dstPath)
		fmt.Fprintf(w, "http://%s/download/%s", r.Host, id)
		return
	}
}

// ===============================
// Download Page (same full-screen style)
// ===============================
func downloadPage(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)
	v, ok := fileMap.Load(id)
	if !ok {
		http.Error(w, "File not found or expired", http.StatusNotFound)
		return
	}
	filePath := v.(string)
	filename := originalFilename(filePath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>f2p • Download %s</title>
<style>
html,body {
  height:100%%; margin:0; background:#000; color:#00ff66;
  font-family:"Courier New", monospace;
  display:flex; flex-direction:column;
  justify-content:center; align-items:center;
  text-align:center;
}
h2 { font-weight:normal; font-size:1.6rem; margin-bottom:1rem; }
p { word-break:break-all; margin-bottom:1rem; }
button {
  background:#00ff66; color:#000; border:none;
  padding:0.7rem 1.6rem; cursor:pointer;
  font-family:inherit; border-radius:4px;
  font-size:1rem; width:60%%; max-width:300px;
}
button:hover { background:#00cc55; }
footer {
  position:fixed; bottom:1rem; font-size:0.8rem; color:#008844;
}
</style>
</head>
<body>
<h2>Ready to download:</h2>
<p>%s</p>
<button onclick="startDownload()">⬇ Download</button>
<footer>f2p • file-to-peer</footer>
<script>
function startDownload(){
  window.location='/file/%s';
  const b=document.querySelector('button');
  b.disabled=true;
  b.textContent='Downloading...';
}
</script>
</body></html>`, filename, filename, id)
}

// ===============================
// Download Handler
// ===============================
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)
	v, ok := fileMap.Load(id)
	if !ok {
		http.Error(w, "File not found or expired", http.StatusNotFound)
		return
	}
	filePath := v.(string)
	fileMap.Delete(id)

	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File missing", http.StatusGone)
		return
	}
	defer f.Close()

	info, _ := f.Stat()
	filename := originalFilename(filePath)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	if _, err := io.Copy(w, f); err != nil {
		log.Printf("Download interrupted: %v", err)
	} else {
		log.Printf("File %s downloaded successfully", filePath)
	}
	f.Close()
	os.Remove(filePath)
	log.Printf("File %s deleted", filePath)
}

// ===============================
// Helpers
// ===============================
func randomID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func originalFilename(path string) string {
	base := filepath.Base(path)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return base
}
