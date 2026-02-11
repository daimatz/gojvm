public class StringConcat {
    public static void main(String[] args) {
        // Use variables to prevent compile-time constant folding
        String hello = "Hello";
        String world = "World";
        String s = hello + " " + world;
        System.out.println(s);

        int n = 42;
        String t = "x=" + n;
        System.out.println(t);

        int a = 3, b = 4, c = 7;
        String u = "" + a + "+" + b + "=" + c;
        System.out.println(u);
    }
}
