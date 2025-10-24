package main

import (
	"fmt"
	"net/http"
)

func uploadPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>P2P File Share</title>
<style>
body {
	font-family: system-ui, sans-serif;
	background: #f6f8fa;
	display: flex;
	justify-content: center;
	align-items: center;
	height: 100vh;
}
.container {
	background: white;
	padding: 2rem;
	border-radius: 10px;
	box-shadow: 0 4px 10px rgba(0,0,0,0.1);
	width: 400px;
	text-align: center;
}
input[type="file"] {
	margin-top: 1rem;
}
button {
	background: #007bff;
	color: white;
	padding: 0.5rem 1rem;
	border: none;
	border-radius: 5px;
	cursor: pointer;
	margin-top: 1rem;
}
button:hover {
	background: #0056b3;
}
#status {
	margin-top: 1rem;
	font-weight: bold;
}
a {
	word-break: break-all;
}
</style>
</head>
<body>
	<div class="container">
		<h2>📤 Upload File</h2>
		<div id="recvLink">Generating link...</div>
		<form id="uploadForm">
			<input type="file" name="senderfile" id="fileInput" required />
			<br/>
			<button type="submit">Upload</button>
		</form>
		<div id="status"></div>
	</div>

<script>
let recvURL = "";
let id = "";

async function initSession() {
	const resp = await fetch("/id");
	recvURL = (await resp.text()).trim();
	id = new URL(recvURL).searchParams.get("id");
	document.getElementById("recvLink").innerHTML = "🔗 Receiver link:<br><a href='" + recvURL + "' target='_blank'>" + recvURL + "</a>";
}

initSession();

const form = document.getElementById("uploadForm");
const statusDiv = document.getElementById("status");

form.addEventListener("submit", async (e) => {
	e.preventDefault();
	const fileInput = document.getElementById("fileInput");
	if (!fileInput.files.length) return;
	
	const formData = new FormData();
	formData.append("senderfile", fileInput.files[0]);

	statusDiv.innerText = "⏫ Uploading...";
	try {
		const uploadResp = await fetch("/upload?id=" + id, {
			method: "POST",
			body: formData
		});
		if (uploadResp.ok) {
			statusDiv.innerText = "✅ Upload complete!";
		} else {
			statusDiv.innerText = "❌ Upload failed: " + uploadResp.statusText;
		}
	} catch (err) {
		statusDiv.innerText = "⚠️ Upload aborted or connection lost.";
	}
});
</script>
</body>
</html>
`)
}
