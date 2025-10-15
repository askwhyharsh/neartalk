import { useState, useEffect, useRef } from "react";
import { MessageCircle, Radio, Users, AlertCircle, Edit2, Check, X, Send } from "lucide-react";

function App() {
  const [sessionId, setSessionId] = useState("");
  const [username, setUsername] = useState("");
  const [tempUsername, setTempUsername] = useState("");
  const [isEditingUsername, setIsEditingUsername] = useState(false);
  const [messages, setMessages] = useState([]);
  const [messageInput, setMessageInput] = useState("");
  const [radius, setRadius] = useState(500);
  const [lat, setLat] = useState(null);
  const [lng, setLng] = useState(null);
  const [nearbyCount, setNearbyCount] = useState(0);
  const [connectionStatus, setConnectionStatus] = useState("disconnected");
  const [error, setError] = useState("");
  
  const wsRef = useRef(null);
  const messagesEndRef = useRef(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    if (!navigator.geolocation) {
      setError("Geolocation is not supported by your browser.");
      return;
    }

    navigator.geolocation.getCurrentPosition(
      (position) => {
        setLat(position.coords.latitude);
        setLng(position.coords.longitude);
      },
      (err) => {
        setError(`Failed to get location: ${err.message}`);
      },
      { enableHighAccuracy: true, timeout: 10000 }
    );
  }, []);

  useEffect(() => {
    if (lat !== null && lng !== null && !sessionId) {
      createSession();
    }
  }, [lat, lng]);

  useEffect(() => {
    // if (!navigator.geolocation) return;

    // const watchId = navigator.geolocation.watchPosition(
    //   (position) => {
    //     setLat(position.coords.latitude);
    //     setLng(position.coords.longitude);
    //   },
    //   (err) => console.error("Location watch error:", err),
    //   { enableHighAccuracy: true, maximumAge: 30000 }
    // );

    // return () => navigator.geolocation.clearWatch(watchId);
    // 18, 73 -> pune
    setLat(18.664998836621084)
    setLng(73.83601479366986)
  }, []);

  useEffect(() => {
    if (sessionId && lat !== null && lng !== null) {
      updateLocation();
    }
  }, [radius]);

  const createSession = async () => {
    try {
      const res = await fetch("/api/session/create", { method: "POST" });
      const data = await res.json();
      if (data.success) {
        const sid = data.data.id;
        const uname = data.data.username;
        setSessionId(sid);
        setUsername(uname);
        setTempUsername(uname);
        
        // Call location update after session creation
        await updateLocationWithSession(sid);
      } else {
        setError("Failed to create session");
      }
    } catch (err) {
      setError("Network error creating session");
    }
  };

  const updateLocationWithSession = async (sid) => {
    if (!sid || lat === null || lng === null) return;

    try {
      const res = await fetch("/api/location/update", {
        method: "POST",
        headers: { "Content-Type": "application/json" , "X-Session-ID": sid},
        body: JSON.stringify({
          session_id: sid,
          latitude: lat,
          longitude: lng,
          radius: radius,
        }),
      });

      const data = await res.json();
      if (data.success) {
        connectWebSocket(sid);
        fetchNearbyCount(sid);
      } else {
        setError("Failed to update location");
      }
    } catch (err) {
      setError("Network error updating location");
    }
  };

  const updateUsernameAPI = async (sid, name) => {
    try {
      const res = await fetch("/api/session/username", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          session_id: sid,
          username: name,
        }),
      });
      const data = await res.json();
      return data.success;
    } catch (err) {
      console.error("Failed to update username:", err);
      return false;
    }
  };

  const handleUsernameEdit = () => {
    setTempUsername(username);
    setIsEditingUsername(true);
  };

  const handleUsernameSave = async () => {
    if (!tempUsername.trim()) {
      setError("Username cannot be empty");
      return;
    }

    const success = await updateUsernameAPI(sessionId, tempUsername);
    if (success) {
      setUsername(tempUsername);
      setIsEditingUsername(false);
      setError("");
    } else {
      setError("Failed to update username");
    }
  };

  const handleUsernameCancel = () => {
    setTempUsername(username);
    setIsEditingUsername(false);
  };

  const updateLocation = async () => {
    if (!sessionId || lat === null || lng === null) return;

    try {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }

      const res = await fetch("/api/location/update", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          session_id: sessionId,
          latitude: lat,
          longitude: lng,
          radius: radius,
        }),
      });

      const data = await res.json();
      if (data.success) {
        connectWebSocket(sessionId);
        fetchNearbyCount(sessionId);
      } else {
        setError("Failed to update location");
      }
    } catch (err) {
      setError("Network error updating location");
    }
  };

  const connectWebSocket = (sid) => {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws?session_id=${sid}`;
    
    const ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      setConnectionStatus("connected");
      setError("");
      fetchRecentMessages(sid); // Add this line
      const pingInterval = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }));
        }
      }, 30000);
      ws.pingInterval = pingInterval;
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        
        if (msg.type === "chat_message") {
          setMessages((prev) => [...prev, msg]);
        } else if (msg.type === "user_joined") {
          if (msg.user_count !== undefined) {
            setNearbyCount(msg.user_count);
          }
        } else if (msg.type === "user_left") {
          if (msg.user_count !== undefined) {
            setNearbyCount(msg.user_count);
          }
        } else if (msg.type === "error") {
          setError(msg.content || "An error occurred");
        }
      } catch (err) {
        console.error("Failed to parse message:", err);
      }
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
      setConnectionStatus("error");
      setError("Connection error");
    };

    ws.onclose = () => {
      setConnectionStatus("disconnected");
      if (ws.pingInterval) clearInterval(ws.pingInterval);
      
      setTimeout(() => {
        if (sessionId && lat !== null && lng !== null) {
          connectWebSocket(sessionId);
        }
      }, 5000);
    };

    wsRef.current = ws;
  };

  const fetchNearbyCount = async (sid) => {
    if (!sid) return;
    
    try {
      const res = await fetch(`/api/nearby?session_id=${sid}`);
      const data = await res.json();
      if (data.success) {
        setNearbyCount(data.data.count || 0);
      }
    } catch (err) {
      console.error("Failed to fetch nearby count:", err);
    }
  };

  const fetchRecentMessages = async (sid) => {
    if (!sid) return;
    
    try {
      const res = await fetch(`/api/recent-messages?session_id=${sid}`);
      const data = await res.json();
      if (data.success && data.data.messages) {
        setMessages(data.data.messages.reverse()); // Reverse to show oldest first
      }
    } catch (err) {
      console.error("Failed to fetch recent messages:", err);
    }
  };

  const sendMessage = () => {
    if (!messageInput.trim()) return;
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setError("Not connected to chat");
      return;
    }

    
    const message = {
      type: "chat_message",
      content: messageInput.trim(),
      timestamp: Math.floor(Date.now() / 1000)
    };
    console.log("sending message", message)
    try {
      wsRef.current.send(JSON.stringify(message));
      console.log("sent")
      
    } catch (error) {
      console.log("send message err", error)
    }
    setMessageInput("");
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const formatDistance = (meters) => {
    if (!meters) return "";
    const m = parseFloat(meters);
    if (m < 1000) {
      return `${Math.round(m)}m`;
    }
    return `${(m / 1000).toFixed(1)}km`;
  };

  const formatTime = (timestamp) => {
    const date = new Date(timestamp * 1000);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex flex-col">
      <div className="bg-slate-800/80 backdrop-blur-sm border-b border-slate-700 px-4 py-3 shadow-lg">
        <div className="max-w-4xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Radio className={`w-6 h-6 ${connectionStatus === 'connected' ? 'text-green-400 animate-pulse' : 'text-slate-500'}`} />
            <div>
              <h1 className="text-xl font-bold text-white">NearTalk</h1>
              <p className="text-xs text-slate-400">
                {connectionStatus === 'connected' ? 'Live' : 'Connecting...'}
              </p>
            </div>
          </div>
          
          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-2 text-slate-300">
              <Users className="w-4 h-4" />
              <span>{nearbyCount}</span>
            </div>
            <div className="flex items-center gap-2">
              {isEditingUsername ? (
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={tempUsername}
                    onChange={(e) => setTempUsername(e.target.value)}
                    className="bg-slate-700 text-white px-2 py-1 rounded text-sm w-32 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    maxLength={20}
                    autoFocus
                  />
                  <button
                    onClick={handleUsernameSave}
                    className="text-green-400 hover:text-green-300"
                    title="Save"
                  >
                    <Check className="w-4 h-4" />
                  </button>
                  <button
                    onClick={handleUsernameCancel}
                    className="text-red-400 hover:text-red-300"
                    title="Cancel"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              ) : (
                <div className="flex items-center gap-2 text-slate-300">
                  <span>{username}</span>
                  <button
                    onClick={handleUsernameEdit}
                    className="text-slate-400 hover:text-slate-200"
                    title="Edit username"
                  >
                    <Edit2 className="w-4 h-4" />
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/50 border-b border-red-800 px-4 py-2">
          <div className="max-w-4xl mx-auto flex items-center gap-2 text-red-200 text-sm">
            <AlertCircle className="w-4 h-4" />
            {error}
          </div>
        </div>
      )}

      <div className="flex-1 flex flex-col max-w-4xl w-full mx-auto">
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {messages.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-slate-400">
              <MessageCircle className="w-16 h-16 mb-4 opacity-50" />
              <p className="text-lg">No messages yet</p>
              <p className="text-sm">Start a conversation with people nearby</p>
            </div>
          ) : (
            messages.map((msg) => (
              <div
                key={msg.id || msg.timestamp}
                className={`flex ${msg.sender_id === sessionId ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                    msg.sender_id === sessionId
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-100'
                  }`}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-semibold text-sm">
                      {msg.sender_id === sessionId ? 'You' : msg.username}
                    </span>
                    {msg.distance && (
                      <span className="text-xs opacity-75">
                        {formatDistance(msg.distance)}
                      </span>
                    )}
                  </div>
                  <p className="text-sm break-words">{msg.content}</p>
                  <div className="text-xs opacity-75 mt-1">
                    {formatTime(msg.timestamp)}
                  </div>
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>

        <div className="border-t border-slate-700 bg-slate-800/50 p-4">
          <div className="flex items-center gap-2 mb-3">
            <label className="text-slate-300 text-sm">Radius:</label>
            <input
              type="range"
              min="100"
              max="5000"
              step="100"
              value={radius}
              onChange={(e) => setRadius(parseInt(e.target.value))}
              className="flex-1"
              disabled={!sessionId}
            />
            <span className="text-slate-300 text-sm w-16">
              {radius < 1000 ? `${radius}m` : `${(radius / 1000).toFixed(1)}km`}
            </span>
          </div>

          <div className="flex gap-2">
            <input
              type="text"
              value={messageInput}
              onChange={(e) => setMessageInput(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder="Type a message..."
              className="flex-1 bg-slate-700 text-white px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={connectionStatus !== 'connected'}
            />
            <button
              onClick={sendMessage}
              disabled={connectionStatus !== 'connected' || !messageInput.trim()}
              className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white px-4 py-2 rounded-lg transition-colors"
            >
              <Send className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;