interface Calc {
    int compute(int a, int b);
}
class Adder implements Calc {
    public int compute(int a, int b) { return a + b; }
}
class Multiplier implements Calc {
    public int compute(int a, int b) { return a * b; }
}
class Interface {
    static int apply(Calc c, int a, int b) {
        return c.compute(a, b);
    }
    public static void main(String[] args) {
        System.out.println(apply(new Adder(), 3, 4));
        System.out.println(apply(new Multiplier(), 3, 4));
    }
}
