public class StaticInit {
    static int counter = 0;
    static String label;

    static {
        counter = 10;
        label = "hello";
    }

    static int increment() {
        counter++;
        return counter;
    }

    public static void main(String[] args) {
        System.out.println(counter);        // 10
        System.out.println(label);          // hello
        System.out.println(increment());    // 11
        System.out.println(increment());    // 12
        System.out.println(counter);        // 12
    }
}
