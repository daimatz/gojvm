public class GenericClass {
    static class Pair<A, B> {
        A first;
        B second;
        Pair(A a, B b) { this.first = a; this.second = b; }
        A getFirst() { return first; }
        B getSecond() { return second; }
        public String toString() { return "(" + first + ", " + second + ")"; }
    }

    static <T> T firstNonNull(T a, T b) {
        return a != null ? a : b;
    }

    public static void main(String[] args) {
        Pair<String, Integer> p = new Pair<>("hello", 42);
        System.out.println(p.getFirst());     // hello
        System.out.println(p.getSecond());    // 42
        System.out.println(p.toString());     // (hello, 42)

        String s = firstNonNull(null, "default");
        System.out.println(s);               // default

        Pair<Integer, Integer> p2 = new Pair<>(10, 20);
        System.out.println(p2.getFirst() + p2.getSecond()); // 30
    }
}
