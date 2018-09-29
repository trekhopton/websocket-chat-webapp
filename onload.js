window.onload = function () {
    var conn;
    var msgInput = document.getElementById("msgInput");
    var log = document.getElementById("log");

    var username = sessionStorage.getItem("username");
    if (!username) {
        window.location.href = window.location.href.replace('/chatroom.html','');
        return false
    }
    
    function appendLog(item) {
        var doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
        log.appendChild(item);
        if (doScroll) {
            log.scrollTop = log.scrollHeight - log.clientHeight;
        }
    }

    document.getElementById("form").onsubmit = function () {
        if (!conn) {
            return false;
        }
        if (!msgInput.value) {
            return false;
        }
        var msg = {username: username, text: msgInput.value};
        conn.send(JSON.stringify(msg));
        msgInput.value = "";
        return false;
    };

    // check if browser supports websockets
    if (window["WebSocket"]) {
        conn = new WebSocket("ws://" + document.location.host + "/ws");
        conn.onclose = function (evt) {
            var item = document.createElement("div");
            item.innerHTML = "<b>Connection closed.</b>";
            appendLog(item);
        };

        conn.onopen = function(evt) {
            var joinMsg = {type: "join", username: username};
            conn.send(JSON.stringify(joinMsg));
        }

        conn.onmessage = function (evt) {
            var messages = evt.data.split('\n');
            for (var i = 0; i < messages.length; i++) {
                console.log(messages[i]);
                var msgObj = JSON.parse(messages[i]);

                var card = document.createElement("div");
                card.title = "UserID: "+msgObj.userID;
                card.className = "card";
                
                var cardBody = document.createElement("div");
                cardBody.className = "card-body";
                card.appendChild(cardBody);

                var cardSub = document.createElement("h8");
                cardSub.className = "card-subtitle text-muted";
                cardSub.innerText = msgObj.username; 
                cardBody.appendChild(cardSub);

                var cardText = document.createElement("p");
                cardText.className = "card-text";
                cardText.innerText = msgObj.text; 
                cardBody.appendChild(cardText);

                appendLog(card);
            }
        };
    } else {
        var item = document.createElement("div");
        item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
        appendLog(item);
    }
};