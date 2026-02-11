public class DefaultMethod {
    interface Greeter {
        String greet(String name);
        // Java 8 default method
        default String shout(String name) {
            return greet(name).toUpperCase();
        }
    }

    interface Logger {
        default String log(String msg) {
            return "[LOG] " + msg;
        }
    }

    // Implement multiple interfaces with default methods
    static class FriendlyGreeter implements Greeter, Logger {
        public String greet(String name) {
            return "Hello, " + name + "!";
        }
    }

    // Override default method
    static class FormalGreeter implements Greeter {
        public String greet(String name) {
            return "Good day, " + name + ".";
        }
        public String shout(String name) {
            return "ATTENTION: " + name;
        }
    }

    public static void main(String[] args) {
        FriendlyGreeter fg = new FriendlyGreeter();
        System.out.println(fg.greet("Alice"));    // Hello, Alice!
        System.out.println(fg.shout("Alice"));    // HELLO, ALICE!
        System.out.println(fg.log("test"));       // [LOG] test

        FormalGreeter formal = new FormalGreeter();
        System.out.println(formal.greet("Bob"));  // Good day, Bob.
        System.out.println(formal.shout("Bob"));  // ATTENTION: Bob
    }
}
