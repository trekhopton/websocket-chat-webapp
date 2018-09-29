window.onload = function () {
    console.log("login page loaded")
    var username = document.getElementById("username");

    document.getElementById("usernameForm").onsubmit = function () {
        if(!username.value) {
            console.log("no username entered")
            return false;
        }
        usr = username.value.trim();
        var validChars = /^[0-9a-zA-Z_]+$/;
        if(!usr.match(validChars) || usr.length < 3 || usr.length > 20 || usr == "SYSTEM") {
            console.log("invalid username")
            alert("Invalid username entered.\n\nUsername must:\n\n\u2022 only contain letters, numbers or underscores\n\u2022 be 3 to 20 characters long\n\u2022 not be SYSTEM");
            return false;
        }
        console.log("username entered: "+usr);
        sessionStorage.setItem("username", usr);
        window.location.href += "/chatroom.html";
        return false;
    };
};