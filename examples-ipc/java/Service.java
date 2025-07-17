import java.io.*;
import java.net.Socket;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.text.SimpleDateFormat;
import java.util.*;
import java.util.concurrent.*;
import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;

public class Service {
    private static final byte REQUEST = 1;
    private static final byte RESPONSE = 2;
    private static final byte REGISTER = 3;
    private static final byte HEARTBEAT = 4;
    
    private final Socket socket;
    private final DataInputStream input;
    private final DataOutputStream output;
    private final String serviceName = "demo-service-java";
    private final Map<String, Handler> handlers = new HashMap<>();
    private final Gson gson = new Gson();
    private final ExecutorService executor = Executors.newCachedThreadPool();
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);
    
    public interface Handler {
        Object handle(Map<String, Object> params) throws Exception;
    }
    
    public Service(String host, int port) throws IOException {
        this.socket = new Socket(host, port);
        this.input = new DataInputStream(socket.getInputStream());
        this.output = new DataOutputStream(socket.getOutputStream());
        System.out.println("Connected to Neo IPC server at " + host + ":" + port);
    }
    
    public void registerService(Map<String, String> metadata) throws IOException {
        Map<String, Object> registerData = new HashMap<>();
        registerData.put("name", serviceName);
        registerData.put("metadata", metadata);
        
        byte[] data = gson.toJson(registerData).getBytes();
        sendMessage(REGISTER, "", serviceName, "", new HashMap<>(), data);
        System.out.println("Service '" + serviceName + "' registered");
    }
    
    public void addHandler(String method, Handler handler) {
        handlers.put(method, handler);
        System.out.println("Handler registered for method: " + method);
    }
    
    private void sendMessage(byte type, String id, String service, String method,
                           Map<String, String> metadata, byte[] data) throws IOException {
        ByteArrayOutputStream content = new ByteArrayOutputStream();
        
        // 消息类型
        content.write(type);
        
        // ID
        byte[] idBytes = id.getBytes();
        writeInt(content, idBytes.length);
        content.write(idBytes);
        
        // Service
        byte[] serviceBytes = service.getBytes();
        writeInt(content, serviceBytes.length);
        content.write(serviceBytes);
        
        // Method
        byte[] methodBytes = method.getBytes();
        writeInt(content, methodBytes.length);
        content.write(methodBytes);
        
        // Metadata
        byte[] metadataBytes = gson.toJson(metadata).getBytes();
        writeInt(content, metadataBytes.length);
        content.write(metadataBytes);
        
        // Data
        writeInt(content, data.length);
        content.write(data);
        
        // 发送消息
        byte[] contentBytes = content.toByteArray();
        synchronized (output) {
            writeInt(output, contentBytes.length);
            output.write(contentBytes);
            output.flush();
        }
    }
    
    private void writeInt(OutputStream out, int value) throws IOException {
        ByteBuffer buffer = ByteBuffer.allocate(4);
        buffer.order(ByteOrder.LITTLE_ENDIAN);
        buffer.putInt(value);
        out.write(buffer.array());
    }
    
    private int readInt(InputStream in) throws IOException {
        byte[] bytes = new byte[4];
        in.read(bytes);
        ByteBuffer buffer = ByteBuffer.wrap(bytes);
        buffer.order(ByteOrder.LITTLE_ENDIAN);
        return buffer.getInt();
    }
    
    private Message readMessage() throws IOException {
        // 读取消息长度
        int msgLen = readInt(input);
        
        // 读取消息内容
        byte[] msgData = new byte[msgLen];
        input.readFully(msgData);
        
        ByteArrayInputStream bais = new ByteArrayInputStream(msgData);
        
        // 解析消息
        Message msg = new Message();
        msg.type = (byte) bais.read();
        
        // ID
        int idLen = readInt(bais);
        byte[] idBytes = new byte[idLen];
        bais.read(idBytes);
        msg.id = new String(idBytes);
        
        // Service
        int serviceLen = readInt(bais);
        byte[] serviceBytes = new byte[serviceLen];
        bais.read(serviceBytes);
        msg.service = new String(serviceBytes);
        
        // Method
        int methodLen = readInt(bais);
        byte[] methodBytes = new byte[methodLen];
        bais.read(methodBytes);
        msg.method = new String(methodBytes);
        
        // Metadata
        int metadataLen = readInt(bais);
        byte[] metadataBytes = new byte[metadataLen];
        bais.read(metadataBytes);
        if (metadataLen > 0) {
            msg.metadata = gson.fromJson(new String(metadataBytes), 
                new TypeToken<Map<String, String>>(){}.getType());
        } else {
            msg.metadata = new HashMap<>();
        }
        
        // Data
        int dataLen = readInt(bais);
        msg.data = new byte[dataLen];
        bais.read(msg.data);
        
        return msg;
    }
    
    private void handleRequest(Message msg) {
        Handler handler = handlers.get(msg.method);
        if (handler == null) {
            try {
                Map<String, String> errorMeta = new HashMap<>();
                errorMeta.put("error", "true");
                String errorData = String.format("{\"error\":\"Method '%s' not found\"}", msg.method);
                sendMessage(RESPONSE, msg.id, msg.service, msg.method, errorMeta, errorData.getBytes());
            } catch (IOException e) {
                e.printStackTrace();
            }
            return;
        }
        
        try {
            // 解析请求参数
            Map<String, Object> params = new HashMap<>();
            if (msg.data.length > 0) {
                params = gson.fromJson(new String(msg.data), 
                    new TypeToken<Map<String, Object>>(){}.getType());
            }
            
            // 调用处理器
            Object result = handler.handle(params);
            
            // 发送响应
            byte[] responseData = gson.toJson(result).getBytes();
            sendMessage(RESPONSE, msg.id, msg.service, msg.method, new HashMap<>(), responseData);
            
        } catch (Exception e) {
            try {
                Map<String, String> errorMeta = new HashMap<>();
                errorMeta.put("error", "true");
                String errorData = String.format("{\"error\":\"%s\"}", e.getMessage());
                sendMessage(RESPONSE, msg.id, msg.service, msg.method, errorMeta, errorData.getBytes());
            } catch (IOException ex) {
                ex.printStackTrace();
            }
        }
    }
    
    private void startHeartbeat() {
        scheduler.scheduleAtFixedRate(() -> {
            try {
                sendMessage(HEARTBEAT, "", serviceName, "", new HashMap<>(), new byte[0]);
                System.out.println("Heartbeat sent");
            } catch (IOException e) {
                e.printStackTrace();
            }
        }, 30, 30, TimeUnit.SECONDS);
    }
    
    public void run() throws IOException {
        startHeartbeat();
        
        while (!socket.isClosed()) {
            Message msg = readMessage();
            if (msg.type == REQUEST) {
                executor.execute(() -> handleRequest(msg));
            }
        }
    }
    
    private static class Message {
        byte type;
        String id;
        String service;
        String method;
        Map<String, String> metadata;
        byte[] data;
    }
    
    public static void main(String[] args) throws Exception {
        // 从环境变量读取配置
        String host = System.getenv("NEO_IPC_HOST");
        if (host == null) host = "localhost";
        
        String portStr = System.getenv("NEO_IPC_PORT");
        int port = portStr != null ? Integer.parseInt(portStr) : 9999;  // 使用正确的默认端口
        
        // 创建服务
        Service service = new Service(host, port);
        
        // 注册处理器
        service.addHandler("hello", params -> {
            String name = (String) params.getOrDefault("name", "World");
            Map<String, Object> response = new HashMap<>();
            response.put("message", "Hello, " + name + "!");
            response.put("timestamp", new Date().toString());
            response.put("service", "Java Demo Service");
            return response;
        });
        
        service.addHandler("calculate", params -> {
            double a = ((Number) params.getOrDefault("a", 0)).doubleValue();
            double b = ((Number) params.getOrDefault("b", 0)).doubleValue();
            String operation = (String) params.getOrDefault("operation", "add");
            
            Object result;
            switch (operation) {
                case "add":
                    result = a + b;
                    break;
                case "subtract":
                    result = a - b;
                    break;
                case "multiply":
                    result = a * b;
                    break;
                case "divide":
                    result = b != 0 ? a / b : "Cannot divide by zero";
                    break;
                default:
                    result = "Unknown operation";
            }
            
            Map<String, Object> response = new HashMap<>();
            response.put("result", result);
            response.put("operation", operation);
            response.put("a", a);
            response.put("b", b);
            return response;
        });
        
        service.addHandler("echo", params -> {
            String message = (String) params.getOrDefault("message", "");
            Map<String, Object> response = new HashMap<>();
            response.put("echo", message);
            response.put("length", message.length());
            response.put("reversed", new StringBuilder(message).reverse().toString());
            return response;
        });
        
        service.addHandler("getTime", params -> {
            String format = (String) params.getOrDefault("format", "iso");
            Date now = new Date();
            
            String timeStr;
            switch (format) {
                case "unix":
                    timeStr = String.valueOf(now.getTime() / 1000);
                    break;
                case "readable":
                    timeStr = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss").format(now);
                    break;
                default:
                    timeStr = now.toString();
            }
            
            Map<String, Object> response = new HashMap<>();
            response.put("time", timeStr);
            response.put("timezone", TimeZone.getDefault().getID());
            response.put("format", format);
            return response;
        });
        
        service.addHandler("getInfo", params -> {
            Map<String, Object> response = new HashMap<>();
            response.put("service", "demo-service-java");
            response.put("language", "Java");
            response.put("version", "1.0.0");
            response.put("handlers", new ArrayList<>(service.handlers.keySet()));
            response.put("uptime", "N/A");
            
            Map<String, String> system = new HashMap<>();
            system.put("platform", "java");
            system.put("java_version", System.getProperty("java.version"));
            response.put("system", system);
            
            return response;
        });
        
        // 注册服务
        Map<String, String> metadata = new HashMap<>();
        metadata.put("language", "java");
        metadata.put("version", "1.0.0");
        metadata.put("description", "Java demo service for Neo Framework");
        service.registerService(metadata);
        
        System.out.println("Java demo service is running...");
        System.out.println("Listening on " + host + ":" + port);
        System.out.println("Available methods: hello, calculate, echo, getTime, getInfo");
        
        // 运行服务
        service.run();
    }
}