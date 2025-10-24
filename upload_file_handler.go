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
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        background: linear-gradient(135deg, #1e293b, #0f172a);
        color: #f1f5f9;
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100vh;
        margin: 0;
    }
    .card {
        background: #1e293b;
        padding: 2rem;
        border-radius: 1.25rem;
        text-align: center;
        box-shadow: 0 10px 20px rgba(0,0,0,0.3);
        width: 90%;
        max-width: 500px;
        border: 1px solid #334155;
    }
    h1, h2 {
        font-size: 1.5rem;
        margin-bottom: 1rem;
        color: #f1f5f9;
    }
    p {
        color: #cbd5e1;
        margin-bottom: 1.5rem;
    }
    input[type="file"] {
        margin: 1rem 0;
        color: #e2e8f0;
    }
    .upload-btn {
        display: inline-block;
        background: #3b82f6;
        color: white;
        padding: 0.75rem 1.5rem;
        border-radius: 9999px;
        text-decoration: none;
        font-weight: 600;
        transition: background 0.2s ease-in-out;
        border: none;
        cursor: pointer;
        width: 100%;
        margin-top: 0.5rem;
    }
    .upload-btn:hover {
        background: #2563eb;
    }
    #status {
        margin-top: 1.5rem;
        font-weight: 500;
        min-height: 1.5rem;
    }
    .success {
        color: #4ade80;
    }
    .error {
        color: #f87171;
    }
    .warning {
        color: #fbbf24;
    }
    #recvLink {
        background: #334155;
        padding: 1rem;
        border-radius: 0.5rem;
        margin-bottom: 1.5rem;
        word-break: break-all;
        font-size: 0.875rem;
        text-align: left;
    }
    #copyContainer {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 0.5rem;
        color: #f1f5f9;
    }
    #copyBtn {
        background: #3b82f6;
        color: white;
        padding: 0.25rem 0.75rem;
        border-radius: 9999px;
        border: none;
        cursor: pointer;
        font-size: 0.875rem;
				margin-bottom: 0.5rem;
        transition: background 0.2s ease-in-out;
    }
    #copyBtn:hover {
        background: #2563eb;
    }
    .copied {
        background: #10b981 !important;
    }
    a {
        color: #60a5fa;
        text-decoration: none;
        display: block;
        word-break: break-all;
    }
    a:hover {
        text-decoration: underline;
    }
</style>
</head>
<body>
	<div class="card">
		<h2>📤 Upload File</h2>
		<div id="recvLink">
            <div id="copyContainer">
                <span>🔗 Receiver link:</span>
                <button id="copyBtn">Copy</button>
            </div>
            <div id="linkDisplay">Generating link...</div>
        </div>
		<form id="uploadForm">
			<input type="file" name="senderfile" id="fileInput" required />
			<br/>
			<button type="submit" class="upload-btn">Upload File</button>
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
	document.getElementById("linkDisplay").innerHTML = "<a href='" + recvURL + "' target='_blank'>" + recvURL + "</a>";
}

async function copyLink() {
    try {
        await navigator.clipboard.writeText(recvURL);
        // Show temporary success message
        const copyBtn = document.getElementById("copyBtn");
        const originalText = copyBtn.innerText;
        copyBtn.innerText = "✓ Copied!";
        copyBtn.classList.add("copied");
        
        setTimeout(function() {
            copyBtn.innerText = originalText;
            copyBtn.classList.remove("copied");
        }, 2000);
    } catch (err) {
        console.error("Failed to copy link: ", err);
        alert("Failed to copy link to clipboard");
    }
}

document.getElementById("copyBtn").addEventListener("click", copyLink);

initSession();

const form = document.getElementById("uploadForm");
const statusDiv = document.getElementById("status");

form.addEventListener("submit", async function(e) {
	e.preventDefault();
	const fileInput = document.getElementById("fileInput");
	if (!fileInput.files.length) return;
	
	const formData = new FormData();
	formData.append("senderfile", fileInput.files[0]);

	statusDiv.className = "warning";
	statusDiv.innerText = "⏫ Uploading...";
	try {
		const uploadResp = await fetch("/upload?id=" + id, {
			method: "POST",
			body: formData
		});
		if (uploadResp.ok) {
			statusDiv.className = "success";
			statusDiv.innerText = "✅ Upload complete!";
		} else {
			statusDiv.className = "error";
			statusDiv.innerText = "❌ Upload failed: " + uploadResp.statusText;
		}
	} catch (err) {
		statusDiv.className = "error";
		statusDiv.innerText = "⚠️ Upload aborted or connection lost.";
	}
});
</script>
</body>
</html>
`)
}
